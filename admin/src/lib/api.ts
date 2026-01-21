import axios, { type AxiosError, type AxiosInstance } from 'axios'
import {
  getAccessToken,
  getRefreshToken,
  setTokens,
  clearTokens,
  type AdminUser,
} from './auth'

// Base URL for the API - priority order:
// 1. Runtime config injected by server (FLUXBASE_PUBLIC_BASE_URL)
// 2. Build-time environment variable (VITE_API_URL)
// 3. Empty string (relative URLs) - works when dashboard is served from the same domain
const API_BASE_URL =
  window.__FLUXBASE_CONFIG__?.publicBaseURL ||
  import.meta.env.VITE_API_URL ||
  ''

// Create axios instance with default config
const api: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 30000, // 30 seconds
})

// Request interceptor to add auth token (admin token only)
// Note: If a custom Authorization header is already set (e.g., for impersonation),
// we don't overwrite it - this allows components to pass their own token
api.interceptors.request.use(
  (config) => {
    // Don't overwrite if Authorization header is already set (e.g., impersonation token)
    if (!config.headers.Authorization) {
      const accessToken = getAccessToken()
      if (accessToken) {
        config.headers.Authorization = `Bearer ${accessToken}`
      }
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor to handle token refresh and errors
let isRefreshing = false
let failedQueue: Array<{
  resolve: (value: unknown) => void
  reject: (reason?: unknown) => void
}> = []

const processQueue = (error: Error | null, token: string | null = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error)
    } else {
      prom.resolve(token)
    }
  })

  failedQueue = []
}

// Helper to check if response indicates user is not authenticated
const isNotLoggedInResponse = (data: unknown): boolean => {
  if (!data || typeof data !== 'object') return false
  const obj = data as Record<string, unknown>
  // Check for common error messages indicating authentication issues
  const errorFields = [obj.error, obj.message, obj.msg, obj.detail]
  for (const field of errorFields) {
    if (typeof field === 'string') {
      const lower = field.toLowerCase()
      if (
        lower.includes('not logged in') ||
        lower.includes('not authenticated') ||
        lower.includes('unauthorized') ||
        lower.includes('invalid token') ||
        lower.includes('token expired') ||
        lower.includes('session expired') ||
        lower.includes('authentication required')
      ) {
        return true
      }
    }
  }
  return false
}

api.interceptors.response.use(
  (response) => {
    // Check if successful response contains auth error message
    if (isNotLoggedInResponse(response.data)) {
      const refreshToken = getRefreshToken()
      if (refreshToken) {
        // Try to refresh the token and retry
        return axios
          .post(`${API_BASE_URL}/api/v1/admin/refresh`, {
            refresh_token: refreshToken,
          })
          .then((refreshResponse) => {
            const {
              access_token,
              refresh_token: newRefreshToken,
              user,
              expires_in,
            } = refreshResponse.data
            setTokens(
              { access_token, refresh_token: newRefreshToken, expires_in },
              user as AdminUser
            )
            // Retry the original request with new token
            if (response.config.headers) {
              response.config.headers.Authorization = `Bearer ${access_token}`
            }
            return api(response.config)
          })
          .catch(() => {
            // Refresh failed, redirect to login
            clearTokens()
            window.location.href = '/admin/login'
            return new Promise(() => {})
          })
      }
      // No refresh token, redirect to login
      clearTokens()
      window.location.href = '/admin/login'
      return new Promise(() => {})
    }
    return response
  },
  async (error: AxiosError) => {
    const originalRequest = error.config as typeof error.config & {
      _retry?: boolean
    }

    // Handle 401 Unauthorized - try to refresh token
    if (error.response?.status === 401 && !originalRequest._retry) {
      if (isRefreshing) {
        // If already refreshing, queue this request
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject })
        })
          .then((token) => {
            if (originalRequest.headers) {
              originalRequest.headers.Authorization = `Bearer ${token}`
            }
            return api(originalRequest)
          })
          .catch((err) => {
            return Promise.reject(err)
          })
      }

      originalRequest._retry = true
      isRefreshing = true

      const refreshToken = getRefreshToken()

      if (!refreshToken) {
        // No refresh token, redirect to login
        clearTokens()
        window.location.href = '/admin/login'
        // Return a never-resolving promise to prevent React Query from showing errors
        // while the redirect is happening
        return new Promise(() => {})
      }

      try {
        // Attempt to refresh the token
        const response = await axios.post(
          `${API_BASE_URL}/api/v1/admin/refresh`,
          {
            refresh_token: refreshToken,
          }
        )

        const {
          access_token,
          refresh_token: newRefreshToken,
          user,
          expires_in,
        } = response.data

        // Update tokens
        setTokens(
          { access_token, refresh_token: newRefreshToken, expires_in },
          user as AdminUser
        )

        // Update the failed request and retry
        if (originalRequest.headers) {
          originalRequest.headers.Authorization = `Bearer ${access_token}`
        }

        processQueue(null, access_token)
        isRefreshing = false

        return api(originalRequest)
      } catch (refreshError) {
        processQueue(refreshError as Error, null)
        isRefreshing = false

        // Refresh failed, logout user
        clearTokens()
        window.location.href = '/admin/login'

        // Return a never-resolving promise to prevent React Query from showing errors
        // while the redirect is happening
        return new Promise(() => {})
      }
    }

    // Check for auth error messages in error response body (for non-401 responses)
    if (
      error.response?.data &&
      isNotLoggedInResponse(error.response.data) &&
      !originalRequest._retry
    ) {
      originalRequest._retry = true
      const refreshToken = getRefreshToken()
      if (refreshToken) {
        try {
          const response = await axios.post(
            `${API_BASE_URL}/api/v1/admin/refresh`,
            { refresh_token: refreshToken }
          )
          const {
            access_token,
            refresh_token: newRefreshToken,
            user,
            expires_in,
          } = response.data
          setTokens(
            { access_token, refresh_token: newRefreshToken, expires_in },
            user as AdminUser
          )
          if (originalRequest.headers) {
            originalRequest.headers.Authorization = `Bearer ${access_token}`
          }
          return api(originalRequest)
        } catch {
          // Refresh failed, redirect to login
          clearTokens()
          window.location.href = '/admin/login'
          return new Promise(() => {})
        }
      }
      // No refresh token, redirect to login
      clearTokens()
      window.location.href = '/admin/login'
      return new Promise(() => {})
    }

    return Promise.reject(error)
  }
)

// API Types
export interface User {
  id: string
  email: string
  email_verified: boolean
  role: string
  metadata: Record<string, unknown> | null
  created_at: string
  updated_at: string
}

export interface SignInRequest {
  email: string
  password: string
}

export interface SignInResponse {
  user: User
  access_token: string
  refresh_token: string
  expires_in: number
}

export interface SignUpRequest {
  email: string
  password: string
  metadata?: Record<string, unknown>
}

export interface SignUpResponse {
  user: User
  access_token: string
  refresh_token: string
  expires_in: number
}

// Auth API methods
export const authApi = {
  signIn: async (data: SignInRequest): Promise<SignInResponse> => {
    const response = await api.post<SignInResponse>('/api/v1/auth/signin', data)
    return response.data
  },

  signUp: async (data: SignUpRequest): Promise<SignUpResponse> => {
    const response = await api.post<SignUpResponse>('/api/v1/auth/signup', data)
    return response.data
  },

  signOut: async (): Promise<void> => {
    await api.post('/api/v1/auth/signout')
  },

  getUser: async (): Promise<User> => {
    const response = await api.get<User>('/api/v1/auth/user')
    return response.data
  },

  updateUser: async (
    data: Partial<Pick<User, 'email' | 'metadata'>>
  ): Promise<User> => {
    const response = await api.patch<User>('/api/v1/auth/user', data)
    return response.data
  },

  requestPasswordReset: async (email: string): Promise<{ message: string }> => {
    const response = await api.post<{ message: string }>(
      '/api/v1/auth/password/reset',
      { email }
    )
    return response.data
  },

  resetPassword: async (
    token: string,
    newPassword: string
  ): Promise<{ message: string }> => {
    const response = await api.post<{ message: string }>(
      '/api/v1/auth/password/reset/confirm',
      {
        token,
        new_password: newPassword,
      }
    )
    return response.data
  },

  verifyResetToken: async (
    token: string
  ): Promise<{ valid: boolean; message?: string }> => {
    try {
      const response = await api.post<{ message: string }>(
        '/api/v1/auth/password/reset/verify',
        { token }
      )
      return { valid: true, message: response.data.message }
    } catch {
      return { valid: false, message: 'Invalid or expired token' }
    }
  },
}

// Database API methods
export interface TableInfo {
  schema: string
  name: string
  type: 'table' | 'view' | 'materialized_view'
  rest_path?: string
  columns: Array<{
    name: string
    data_type: string
    is_nullable: boolean
    default_value: string | null
    is_primary_key: boolean
    is_foreign_key: boolean
    is_unique: boolean
    max_length: number | null
    position: number
  }>
  primary_key: string[]
  foreign_keys: unknown
  indexes: unknown
  rls_enabled: boolean
}

export const databaseApi = {
  getSchemas: async (): Promise<string[]> => {
    const response = await api.get<string[]>('/api/v1/admin/schemas')
    return response.data
  },

  getTables: async (schema?: string): Promise<TableInfo[]> => {
    const url = schema
      ? `/api/v1/admin/tables?schema=${encodeURIComponent(schema)}`
      : '/api/v1/admin/tables'
    const response = await api.get<TableInfo[]>(url)
    return response.data
  },

  createSchema: async (
    name: string
  ): Promise<{ success: boolean; schema: string; message: string }> => {
    const response = await api.post('/api/v1/admin/schemas', { name })
    return response.data
  },

  createTable: async (data: {
    schema: string
    name: string
    columns: Array<{
      name: string
      type: string
      nullable: boolean
      primaryKey: boolean
      defaultValue: string
    }>
  }): Promise<{
    success: boolean
    schema: string
    table: string
    message: string
  }> => {
    const response = await api.post('/api/v1/admin/tables', data)
    return response.data
  },

  deleteTable: async (
    schema: string,
    table: string
  ): Promise<{ success: boolean; message: string }> => {
    const response = await api.delete(`/api/v1/admin/tables/${schema}/${table}`)
    return response.data
  },

  renameTable: async (
    schema: string,
    table: string,
    newName: string
  ): Promise<{ success: boolean; message: string }> => {
    const response = await api.patch(
      `/api/v1/admin/tables/${schema}/${table}`,
      { newName }
    )
    return response.data
  },

  addColumn: async (
    schema: string,
    table: string,
    column: {
      name: string
      type: string
      nullable: boolean
      defaultValue?: string
    }
  ): Promise<{ success: boolean; message: string }> => {
    const response = await api.post(
      `/api/v1/admin/tables/${schema}/${table}/columns`,
      column
    )
    return response.data
  },

  dropColumn: async (
    schema: string,
    table: string,
    column: string
  ): Promise<{ success: boolean; message: string }> => {
    const response = await api.delete(
      `/api/v1/admin/tables/${schema}/${table}/columns/${column}`
    )
    return response.data
  },

  getTableData: async <T = unknown>(
    table: string,
    params?: {
      limit?: number
      offset?: number
      order?: string
      select?: string
      filter?: Record<string, unknown>
    }
  ): Promise<T[]> => {
    const response = await api.get<T[]>(`/api/v1/tables/${table}`, { params })
    return response.data
  },

  createRecord: async <T = unknown>(
    table: string,
    data: Record<string, unknown>
  ): Promise<T> => {
    const response = await api.post<T>(`/api/v1/tables/${table}`, data)
    return response.data
  },

  updateRecord: async <T = unknown>(
    table: string,
    id: string | number,
    data: Record<string, unknown>
  ): Promise<T> => {
    const response = await api.patch<T>(`/api/v1/tables/${table}/${id}`, data)
    return response.data
  },

  deleteRecord: async (table: string, id: string | number): Promise<void> => {
    await api.delete(`/api/v1/tables/${table}/${id}`)
  },

  getTableSchema: async (
    table: string
  ): Promise<{
    columns: Array<{
      name: string
      type: string
      nullable: boolean
      default: string | null
      primary_key: boolean
    }>
  }> => {
    const response = await api.get(`/api/admin/tables/${table}/schema`)
    return response.data
  },
}

// Health check
export const healthApi = {
  check: async (): Promise<{
    status: string
    services: { database: boolean; realtime: boolean }
    timestamp: string
  }> => {
    const response = await api.get('/health')
    return response.data
  },
}

// User Management API Types
export interface EnrichedUser {
  id: string
  email: string
  email_verified: boolean
  role: string
  provider: 'email' | 'invite_pending' | 'magic_link'
  active_sessions: number
  last_sign_in: string | null
  is_locked: boolean
  user_metadata: Record<string, unknown> | null
  app_metadata: Record<string, unknown> | null
  created_at: string
  updated_at: string
}

export interface InviteUserRequest {
  email: string
  role: string
  password?: string // Optional: if provided, use this instead of auto-generating
}

export interface InviteUserResponse {
  user: User
  temporary_password?: string
  email_sent: boolean
  message: string
}

// User Management API methods (admin only)
export const userManagementApi = {
  listUsers: async (
    userType: 'app' | 'dashboard' = 'app'
  ): Promise<{ users: EnrichedUser[]; total: number }> => {
    const response = await api.get<{ users: EnrichedUser[]; total: number }>(
      '/api/v1/admin/users',
      {
        params: { type: userType },
      }
    )
    return response.data
  },

  inviteUser: async (
    data: InviteUserRequest,
    userType: 'app' | 'dashboard' = 'app'
  ): Promise<InviteUserResponse> => {
    const response = await api.post<InviteUserResponse>(
      '/api/v1/admin/users/invite',
      data,
      { params: { type: userType } }
    )
    return response.data
  },

  deleteUser: async (
    userId: string,
    userType: 'app' | 'dashboard' = 'app'
  ): Promise<{ message: string }> => {
    const response = await api.delete<{ message: string }>(
      `/api/v1/admin/users/${userId}`,
      { params: { type: userType } }
    )
    return response.data
  },

  updateUserRole: async (
    userId: string,
    role: string,
    userType: 'app' | 'dashboard' = 'app'
  ): Promise<User> => {
    const response = await api.patch<User>(
      `/api/v1/admin/users/${userId}/role`,
      {
        role,
      },
      {
        params: { type: userType },
      }
    )
    return response.data
  },

  resetUserPassword: async (
    userId: string,
    userType: 'app' | 'dashboard' = 'app'
  ): Promise<{ message: string }> => {
    const response = await api.post<{ message: string }>(
      `/api/v1/admin/users/${userId}/reset-password`,
      {},
      { params: { type: userType } }
    )
    return response.data
  },

  updateUser: async (
    userId: string,
    data: {
      email?: string
      role?: string
      password?: string
      user_metadata?: Record<string, unknown>
    },
    userType: 'app' | 'dashboard' = 'app'
  ): Promise<EnrichedUser> => {
    const response = await api.patch<EnrichedUser>(
      `/api/v1/admin/users/${userId}`,
      data,
      { params: { type: userType } }
    )
    return response.data
  },
}

// Edge Functions API Types
export interface EdgeFunction {
  id: string
  name: string
  description?: string
  code: string
  version: number
  cron_schedule?: string
  enabled: boolean
  timeout_seconds: number
  memory_limit_mb: number
  allow_net: boolean
  allow_env: boolean
  allow_read: boolean
  allow_write: boolean
  source: string
  created_at: string
  updated_at: string
}

export interface CreateEdgeFunctionRequest {
  name: string
  description?: string
  code: string
  timeout_seconds: number
  memory_limit_mb: number
  allow_net: boolean
  allow_env: boolean
  allow_read: boolean
  allow_write: boolean
  cron_schedule?: string | null
}

export interface UpdateEdgeFunctionRequest {
  code?: string
  description?: string
  timeout_seconds?: number
  allow_net?: boolean
  allow_env?: boolean
  allow_read?: boolean
  allow_write?: boolean
  cron_schedule?: string | null
  enabled?: boolean
}

export interface EdgeFunctionExecution {
  id: string
  function_id: string
  function_name?: string // Admin view only
  namespace?: string // Admin view only
  trigger_type: string
  status: string
  status_code?: number
  duration_ms?: number
  result?: string
  logs?: string
  error_message?: string
  executed_at: string
  completed_at?: string
}

// Note: FunctionExecutionLog is now stored in the central logging schema (logging.entries)

export interface FunctionReloadResult {
  message?: string
  created?: string[]
  updated?: string[]
  deleted?: string[]
  errors?: string[]
  total?: number
}

export interface FunctionSyncSpec {
  name: string
  description?: string
  code: string
  enabled?: boolean
  timeout_seconds?: number
  memory_limit_mb?: number
  allow_net?: boolean
  allow_env?: boolean
  allow_read?: boolean
  allow_write?: boolean
  allow_unauthenticated?: boolean
  is_public?: boolean
  cron_schedule?: string
}

export interface FunctionSyncOptions {
  namespace?: string
  functions: FunctionSyncSpec[]
  options?: {
    delete_missing?: boolean
    dry_run?: boolean
  }
}

export interface FunctionSyncError {
  function: string
  error: string
  action: 'create' | 'update' | 'delete' | 'bundle'
}

export interface FunctionSyncResult {
  message: string
  namespace: string
  summary: {
    created: number
    updated: number
    deleted: number
    unchanged: number
    errors: number
  }
  details: {
    created: string[]
    updated: string[]
    deleted: string[]
    unchanged: string[]
  }
  errors: FunctionSyncError[]
  dry_run: boolean
}

export interface EdgeFunctionInvokeOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'
  headers?: Record<string, string>
  body?: string
}

// Edge Functions API
export const functionsApi = {
  // List all namespaces with edge functions
  listNamespaces: async (): Promise<string[]> => {
    const response = await api.get<{ namespaces: string[] }>(
      '/api/v1/admin/functions/namespaces'
    )
    return response.data.namespaces || ['default']
  },

  // List all edge functions (optionally filtered by namespace)
  list: async (namespace?: string): Promise<EdgeFunction[]> => {
    const params = namespace ? `?namespace=${namespace}` : ''
    const response = await api.get<EdgeFunction[]>(`/api/v1/functions${params}`)
    return response.data
  },

  // Get a single edge function by name (includes code)
  get: async (name: string): Promise<EdgeFunction> => {
    const response = await api.get<EdgeFunction>(`/api/v1/functions/${name}`)
    return response.data
  },

  // Create edge function
  create: async (data: CreateEdgeFunctionRequest): Promise<EdgeFunction> => {
    const response = await api.post<EdgeFunction>('/api/v1/functions', data)
    return response.data
  },

  // Update edge function
  update: async (
    name: string,
    data: UpdateEdgeFunctionRequest
  ): Promise<EdgeFunction> => {
    const response = await api.put<EdgeFunction>(
      `/api/v1/functions/${name}`,
      data
    )
    return response.data
  },

  // Delete edge function
  delete: async (name: string): Promise<void> => {
    await api.delete(`/api/v1/functions/${name}`)
  },

  // Invoke edge function
  invoke: async (
    name: string,
    options: EdgeFunctionInvokeOptions = {},
    config?: { headers?: Record<string, string> }
  ): Promise<string> => {
    const { method = 'POST', headers = {}, body = '' } = options

    const response = await api.request({
      url: `/api/v1/functions/${name}/invoke`,
      method,
      data: body,
      headers: {
        'Content-Type': 'application/json',
        ...headers,
        ...config?.headers,
      },
      transformResponse: [(data) => data], // Don't parse response, return as string
    })
    return response.data
  },

  // Get execution logs
  getExecutions: async (
    name: string,
    limit = 20
  ): Promise<EdgeFunctionExecution[]> => {
    const response = await api.get<EdgeFunctionExecution[]>(
      `/api/v1/functions/${name}/executions`,
      { params: { limit } }
    )
    return response.data
  },

  // Reload functions from disk (admin only)
  reload: async (): Promise<FunctionReloadResult> => {
    const response = await api.post<FunctionReloadResult>(
      '/api/v1/admin/functions/reload'
    )
    return response.data
  },

  // Sync functions to a namespace (admin only)
  sync: async (options: FunctionSyncOptions): Promise<FunctionSyncResult> => {
    const response = await api.post<FunctionSyncResult>(
      '/api/v1/admin/functions/sync',
      options
    )
    return response.data
  },

  // List all executions across all functions (admin only)
  listAllExecutions: async (filters?: {
    namespace?: string
    function_name?: string
    status?: string
    limit?: number
    offset?: number
  }): Promise<{ executions: EdgeFunctionExecution[]; count: number }> => {
    const params = new URLSearchParams()
    if (filters?.namespace) params.set('namespace', filters.namespace)
    if (filters?.function_name)
      params.set('function_name', filters.function_name)
    if (filters?.status) params.set('status', filters.status)
    if (filters?.limit) params.set('limit', filters.limit.toString())
    if (filters?.offset) params.set('offset', filters.offset.toString())

    const queryString = params.toString()
    const response = await api.get<{
      executions: EdgeFunctionExecution[]
      count: number
    }>(
      `/api/v1/admin/functions/executions${queryString ? `?${queryString}` : ''}`
    )
    return response.data
  },

  // Note: Execution logs are now stored in the central logging schema (logging.entries)
  // Use the logsApi to query execution logs
}

// Jobs API Types
export interface JobFunction {
  id: string
  name: string
  namespace: string
  description?: string
  code?: string
  original_code?: string
  is_bundled: boolean
  bundle_error?: string
  enabled: boolean
  schedule?: string
  timeout_seconds: number
  memory_limit_mb: number
  max_retries: number
  progress_timeout_seconds: number
  allow_net: boolean
  allow_env: boolean
  allow_read: boolean
  allow_write: boolean
  require_role?: string
  version: number
  created_by?: string
  source: string
  created_at: string
  updated_at: string
}

export interface Job {
  id: string
  namespace: string
  job_function_id?: string
  job_name: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'
  payload?: unknown
  result?: unknown
  error_message?: string
  priority: number
  max_duration_seconds?: number
  progress_timeout_seconds?: number
  progress_percent?: number
  progress_message?: string
  progress_data?: unknown
  max_retries: number
  retry_count: number
  worker_id?: string
  created_by?: string
  user_role?: string
  user_email?: string
  user_name?: string
  created_at: string
  started_at?: string
  completed_at?: string
  scheduled_at?: string
  last_progress_at?: string
  /** Estimated completion time (computed, only for running jobs with progress > 0) */
  estimated_completion_at?: string
  /** Estimated seconds remaining (computed, only for running jobs with progress > 0) */
  estimated_seconds_left?: number
}

export type LogLevel = 'debug' | 'info' | 'warning' | 'error' | 'fatal'

// Note: ExecutionLog is now stored in the central logging schema (logging.entries)

export interface JobStats {
  namespace?: string
  pending: number
  running: number
  completed: number
  failed: number
  cancelled: number
  total: number
}

export interface JobWorker {
  id: string
  hostname: string
  status: 'active' | 'idle' | 'dead'
  current_jobs: number
  total_completed: number
  started_at: string
  last_heartbeat_at: string
}

export interface CreateJobFunctionRequest {
  name: string
  namespace?: string
  description?: string
  code: string
  enabled?: boolean
  schedule?: string
  timeout_seconds?: number
  memory_limit_mb?: number
  max_retries?: number
  progress_timeout_seconds?: number
  allow_net?: boolean
  allow_env?: boolean
  allow_read?: boolean
  allow_write?: boolean
}

export interface UpdateJobFunctionRequest {
  description?: string
  code?: string
  enabled?: boolean
  schedule?: string
  timeout_seconds?: number
  memory_limit_mb?: number
  max_retries?: number
  progress_timeout_seconds?: number
  allow_net?: boolean
  allow_env?: boolean
  allow_read?: boolean
  allow_write?: boolean
}

export interface SubmitJobRequest {
  job_name: string
  namespace?: string
  payload?: unknown
  priority?: number
  scheduled?: string
}

export interface JobSyncResult {
  message?: string
  summary: {
    created: number
    updated: number
    deleted: number
    unchanged: number
    errors: number
  }
  functions?: JobFunction[]
  errors?: Array<{ name: string; error: string }>
}

// Jobs API
export const jobsApi = {
  // List all namespaces with job functions
  listNamespaces: async (): Promise<string[]> => {
    const response = await api.get<{ namespaces: string[] }>(
      '/api/v1/admin/jobs/namespaces'
    )
    return response.data.namespaces || ['default']
  },

  // List all job functions (admin view)
  listFunctions: async (namespace?: string): Promise<JobFunction[]> => {
    const params = namespace ? `?namespace=${namespace}` : ''
    const response = await api.get<JobFunction[]>(
      `/api/v1/admin/jobs/functions${params}`
    )
    return response.data
  },

  // Get job function details
  getFunction: async (
    namespace: string,
    name: string
  ): Promise<JobFunction> => {
    const response = await api.get<JobFunction>(
      `/api/v1/admin/jobs/functions/${namespace}/${name}`
    )
    return response.data
  },

  // Create job function
  createFunction: async (
    data: CreateJobFunctionRequest
  ): Promise<JobFunction> => {
    const response = await api.post<JobFunction>(
      '/api/v1/admin/jobs/functions',
      data
    )
    return response.data
  },

  // Update job function
  updateFunction: async (
    namespace: string,
    name: string,
    data: UpdateJobFunctionRequest
  ): Promise<JobFunction> => {
    const response = await api.put<JobFunction>(
      `/api/v1/admin/jobs/functions/${namespace}/${name}`,
      data
    )
    return response.data
  },

  // Delete job function
  deleteFunction: async (namespace: string, name: string): Promise<void> => {
    await api.delete(`/api/v1/admin/jobs/functions/${namespace}/${name}`)
  },

  // Submit job for execution
  submitJob: async (
    data: SubmitJobRequest,
    config?: { headers?: Record<string, string> }
  ): Promise<Job> => {
    const response = await api.post<Job>('/api/v1/jobs/submit', data, config)
    return response.data
  },

  // List jobs (admin view - all jobs)
  listJobs: async (filters?: {
    status?: string
    namespace?: string
    limit?: number
    offset?: number
  }): Promise<Job[]> => {
    const params = new URLSearchParams()
    if (filters?.status) params.append('status', filters.status)
    if (filters?.namespace) params.append('namespace', filters.namespace)
    if (filters?.limit) params.append('limit', filters.limit.toString())
    if (filters?.offset) params.append('offset', filters.offset.toString())

    const queryString = params.toString()
    const response = await api.get<{
      jobs: Job[]
      limit: number
      offset: number
    }>(`/api/v1/admin/jobs/queue${queryString ? `?${queryString}` : ''}`)
    return response.data.jobs
  },

  // Get job details
  getJob: async (jobId: string): Promise<Job> => {
    const response = await api.get<Job>(`/api/v1/admin/jobs/queue/${jobId}`)
    return response.data
  },

  // Note: Execution logs are now stored in the central logging schema (logging.entries)
  // Use the logsApi to query execution logs

  // Cancel job
  cancelJob: async (jobId: string): Promise<void> => {
    await api.post(`/api/v1/admin/jobs/queue/${jobId}/cancel`, {})
  },

  // Terminate job
  terminateJob: async (jobId: string): Promise<void> => {
    await api.post(`/api/v1/admin/jobs/queue/${jobId}/terminate`, {})
  },

  // Retry failed job
  retryJob: async (jobId: string): Promise<Job> => {
    const response = await api.post<Job>(
      `/api/v1/admin/jobs/queue/${jobId}/retry`,
      {}
    )
    return response.data
  },

  // Resubmit job (create new job based on existing one, works for any status)
  resubmitJob: async (jobId: string): Promise<Job> => {
    const response = await api.post<Job>(
      `/api/v1/admin/jobs/queue/${jobId}/resubmit`,
      {}
    )
    return response.data
  },

  // Get job statistics
  getStats: async (namespace?: string): Promise<JobStats> => {
    const params = namespace ? `?namespace=${namespace}` : ''
    const response = await api.get<JobStats>(
      `/api/v1/admin/jobs/stats${params}`
    )
    return response.data
  },

  // List active workers
  listWorkers: async (): Promise<JobWorker[]> => {
    const response = await api.get<JobWorker[]>('/api/v1/admin/jobs/workers')
    return response.data
  },

  // Sync jobs from filesystem
  sync: async (namespace: string): Promise<JobSyncResult> => {
    const response = await api.post<JobSyncResult>('/api/v1/admin/jobs/sync', {
      namespace,
    })
    return response.data
  },
}

// Storage API Types
export interface StorageObject {
  id: string
  bucket: string
  path: string
  mime_type: string
  size: number
  metadata: Record<string, unknown> | null
  owner_id: string | null
  created_at: string
  updated_at: string
}

export interface Bucket {
  id: string
  name: string
  public: boolean
  allowed_mime_types: string[] | null
  max_file_size: number | null
  created_at: string
  updated_at: string
}

export interface BucketListResponse {
  buckets: Bucket[]
}

export interface ObjectListResponse {
  bucket: string
  objects: StorageObject[] | null
  prefixes: string[]
  truncated: boolean
}

// Storage API
export const storageApi = {
  // List all buckets
  listBuckets: async (): Promise<BucketListResponse> => {
    const response = await api.get<BucketListResponse>(
      '/api/v1/storage/buckets'
    )
    return response.data
  },

  // List objects in a bucket
  listObjects: async (
    bucket: string,
    prefix?: string,
    delimiter?: string
  ): Promise<ObjectListResponse> => {
    const params = new URLSearchParams()
    if (prefix) params.append('prefix', prefix)
    if (delimiter) params.append('delimiter', delimiter)

    const response = await api.get<ObjectListResponse>(
      `/api/v1/storage/${bucket}${params.toString() ? `?${params.toString()}` : ''}`
    )
    return response.data
  },

  // Create a bucket
  createBucket: async (bucketName: string): Promise<{ message: string }> => {
    const response = await api.post<{ message: string }>(
      `/api/v1/storage/buckets/${bucketName}`
    )
    return response.data
  },

  // Delete a bucket
  deleteBucket: async (bucketName: string): Promise<{ message: string }> => {
    const response = await api.delete<{ message: string }>(
      `/api/v1/storage/buckets/${bucketName}`
    )
    return response.data
  },

  // Download an object
  downloadObject: async (bucket: string, key: string): Promise<Blob> => {
    const response = await api.get(`/api/v1/storage/${bucket}/${key}`, {
      responseType: 'blob',
    })
    return response.data
  },

  // Delete an object
  deleteObject: async (bucket: string, key: string): Promise<void> => {
    await api.delete(`/api/v1/storage/${bucket}/${key}`)
  },

  // Upload an object (create folder)
  createFolder: async (bucket: string, folderPath: string): Promise<void> => {
    const encodedPath = folderPath
      .split('/')
      .map((segment) => encodeURIComponent(segment))
      .join('/')
    await api.post(`/api/v1/storage/${bucket}/${encodedPath}`, null, {
      headers: { 'Content-Type': 'application/x-directory' },
    })
  },

  // Get object metadata
  getObjectMetadata: async (
    bucket: string,
    key: string
  ): Promise<StorageObject> => {
    const response = await api.get<StorageObject>(
      `/api/v1/storage/${bucket}/${key}`,
      {
        headers: { 'X-Metadata-Only': 'true' },
      }
    )
    return response.data
  },

  // Generate signed URL
  generateSignedUrl: async (
    bucket: string,
    key: string,
    expiresIn: number
  ): Promise<{ url: string; expires_in: number }> => {
    const response = await api.post<{ url: string; expires_in: number }>(
      `/api/v1/storage/${bucket}/${encodeURIComponent(key)}/signed-url`,
      { expires_in: expiresIn }
    )
    return response.data
  },
}

// Webhooks API Types
export interface EventConfig {
  table: string
  operations: string[]
}

export interface WebhookType {
  id: string
  name: string
  description?: string
  url: string
  secret?: string
  enabled: boolean
  events: EventConfig[]
  max_retries: number
  retry_backoff_seconds: number
  timeout_seconds: number
  headers: Record<string, string>
  created_at: string
  updated_at: string
}

export interface WebhookDelivery {
  id: string
  webhook_id: string
  event_type: string
  table_name: string
  record_id?: string
  payload: unknown
  attempt_number: number
  status: string
  http_status_code?: number
  response_body?: string
  error_message?: string
  created_at: string
  delivered_at?: string
}

// Webhooks API
export const webhooksApi = {
  // List all webhooks
  list: async (): Promise<WebhookType[]> => {
    const response = await api.get<WebhookType[]>('/api/v1/webhooks')
    return response.data
  },

  // Get webhook deliveries
  getDeliveries: async (
    webhookId: string,
    limit = 50
  ): Promise<WebhookDelivery[]> => {
    const response = await api.get<WebhookDelivery[]>(
      `/api/v1/webhooks/${webhookId}/deliveries?limit=${limit}`
    )
    return response.data
  },

  // Create webhook
  create: async (webhook: Partial<WebhookType>): Promise<WebhookType> => {
    const response = await api.post<WebhookType>('/api/v1/webhooks', webhook)
    return response.data
  },

  // Update webhook
  update: async (
    id: string,
    updates: Partial<WebhookType>
  ): Promise<WebhookType> => {
    const response = await api.patch<WebhookType>(
      `/api/v1/webhooks/${id}`,
      updates
    )
    return response.data
  },

  // Delete webhook
  delete: async (id: string): Promise<void> => {
    await api.delete(`/api/v1/webhooks/${id}`)
  },

  // Test webhook
  test: async (id: string): Promise<{ message: string }> => {
    const response = await api.post<{ message: string }>(
      `/api/v1/webhooks/${id}/test`
    )
    return response.data
  },
}

// Client Keys API Types
export interface ClientKey {
  id: string
  name: string
  description?: string
  key_prefix: string
  scopes: string[]
  rate_limit_per_minute: number
  last_used_at?: string
  expires_at?: string
  revoked_at?: string
  created_at: string
  updated_at: string
}

export interface CreateClientKeyRequest {
  name: string
  description?: string
  scopes: string[]
  rate_limit_per_minute: number
  expires_at?: string
}

export interface CreateClientKeyResponse {
  client_key: ClientKey
  key: string
}

// Client Keys API
export const clientKeysApi = {
  // List all client keys
  list: async (): Promise<ClientKey[]> => {
    const response = await api.get<ClientKey[]>('/api/v1/client-keys')
    return response.data
  },

  // Create client key
  create: async (
    request: CreateClientKeyRequest
  ): Promise<CreateClientKeyResponse> => {
    const response = await api.post<CreateClientKeyResponse>(
      '/api/v1/client-keys',
      request
    )
    return response.data
  },

  // Revoke client key
  revoke: async (id: string): Promise<{ message: string }> => {
    const response = await api.post<{ message: string }>(
      `/api/v1/client-keys/${id}/revoke`
    )
    return response.data
  },

  // Delete client key
  delete: async (id: string): Promise<{ message: string }> => {
    const response = await api.delete<{ message: string }>(
      `/api/v1/client-keys/${id}`
    )
    return response.data
  },
}

// Backwards compatibility aliases (deprecated)
/** @deprecated Use ClientKey instead */
export type APIKey = ClientKey
/** @deprecated Use CreateClientKeyRequest instead */
export type CreateAPIKeyRequest = CreateClientKeyRequest
/** @deprecated Use CreateClientKeyResponse instead */
export type CreateAPIKeyResponse = CreateClientKeyResponse
/** @deprecated Use clientKeysApi instead */
export const apiKeysApi = clientKeysApi

// Monitoring API Types
export interface SystemMetrics {
  uptime_seconds: number
  go_version: string
  num_goroutines: number
  memory_alloc_mb: number
  memory_total_alloc_mb: number
  memory_sys_mb: number
  num_gc: number
  gc_pause_ms: number
  database: {
    acquire_count: number
    acquired_conns: number
    canceled_acquire_count: number
    constructing_conns: number
    empty_acquire_count: number
    idle_conns: number
    max_conns: number
    total_conns: number
    new_conns_count: number
    max_lifetime_destroy_count: number
    max_idle_destroy_count: number
    acquire_duration_ms: number
  }
  realtime: {
    total_connections: number
    active_channels: number
    total_subscriptions: number
  }
  storage?: {
    total_buckets: number
    total_files: number
    total_size_gb: number
  }
}

export interface HealthStatus {
  status: string
  message?: string
  latency_ms?: number
}

export interface SystemHealth {
  status: string
  services: Record<string, HealthStatus>
}

// Monitoring API
export const monitoringApi = {
  // Get monitoring metrics
  getMetrics: async (): Promise<SystemMetrics> => {
    const response = await api.get<SystemMetrics>('/api/v1/monitoring/metrics')
    return response.data
  },

  // Get health status
  getHealth: async (): Promise<SystemHealth> => {
    const response = await api.get<SystemHealth>('/api/v1/monitoring/health')
    return response.data
  },
}

export default api
export { api as apiClient }

// Admin Authentication API
export const adminAuthAPI = {
  // Check if initial setup is needed
  getSetupStatus: async (): Promise<{
    needs_setup: boolean
    has_admin: boolean
  }> => {
    const response = await axios.get(
      `${API_BASE_URL}/api/v1/admin/setup/status`
    )
    return response.data
  },

  // Initial setup - create first admin user
  initialSetup: async (data: {
    email: string
    password: string
    name: string
    setup_token?: string
  }): Promise<{
    user: AdminUser
    access_token: string
    refresh_token: string
    expires_in: number
  }> => {
    const response = await axios.post(
      `${API_BASE_URL}/api/v1/admin/setup`,
      data
    )
    return response.data
  },

  // Admin login
  login: async (credentials: {
    email: string
    password: string
  }): Promise<{
    user: AdminUser
    access_token: string
    refresh_token: string
    expires_in: number
  }> => {
    const response = await axios.post(
      `${API_BASE_URL}/api/v1/admin/login`,
      credentials
    )
    return response.data
  },

  // Admin logout
  logout: async (): Promise<{ message: string }> => {
    const response = await api.post('/api/v1/admin/logout')
    return response.data
  },

  // Get current admin user
  me: async (): Promise<{ user: AdminUser }> => {
    const response = await api.get('/api/v1/admin/me')
    return response.data
  },
}

// Dashboard user types
export interface DashboardUser {
  id: string
  email: string
  email_verified: boolean
  full_name: string | null
  avatar_url: string | null
  totp_enabled: boolean
  is_active: boolean
  is_locked: boolean
  last_login_at: string | null
  created_at: string
  updated_at: string
  role?: string
}

export interface DashboardSignupRequest {
  email: string
  password: string
  full_name: string
}

export interface DashboardLoginRequest {
  email: string
  password: string
}

export interface DashboardLoginResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  user: DashboardUser
  requires_2fa?: boolean
  user_id?: string
}

export interface UpdateProfileRequest {
  full_name: string
  avatar_url?: string | null
}

export interface ChangePasswordRequest {
  current_password: string
  new_password: string
}

export interface DeleteAccountRequest {
  password: string
}

export interface Setup2FAResponse {
  secret: string
  qr_url: string
}

export interface Enable2FARequest {
  code: string
}

export interface Enable2FAResponse {
  message: string
  backup_codes: string[]
}

export interface Verify2FARequest {
  user_id: string
  code: string
}

export interface Disable2FARequest {
  password: string
}

// Dashboard Auth API methods
export const dashboardAuthAPI = {
  // Signup for dashboard
  signup: async (
    data: DashboardSignupRequest
  ): Promise<{ user: DashboardUser; message: string }> => {
    const response = await axios.post(
      `${API_BASE_URL}/dashboard/auth/signup`,
      data
    )
    return response.data
  },

  // Login to dashboard
  login: async (
    credentials: DashboardLoginRequest
  ): Promise<DashboardLoginResponse> => {
    const response = await axios.post(
      `${API_BASE_URL}/dashboard/auth/login`,
      credentials
    )
    return response.data
  },

  // Get current dashboard user
  me: async (): Promise<DashboardUser> => {
    const response = await api.get('/dashboard/auth/me')
    return response.data
  },

  // Update profile
  updateProfile: async (data: UpdateProfileRequest): Promise<DashboardUser> => {
    const response = await api.put('/dashboard/auth/profile', data)
    return response.data
  },

  // Change password
  changePassword: async (
    data: ChangePasswordRequest
  ): Promise<{ message: string }> => {
    const response = await api.post('/dashboard/auth/password/change', data)
    return response.data
  },

  // Delete account
  deleteAccount: async (
    data: DeleteAccountRequest
  ): Promise<{ message: string }> => {
    const response = await api.delete('/dashboard/auth/account', { data })
    return response.data
  },

  // Setup 2FA
  setup2FA: async (): Promise<Setup2FAResponse> => {
    const response = await api.post('/dashboard/auth/2fa/setup')
    return response.data
  },

  // Enable 2FA
  enable2FA: async (data: Enable2FARequest): Promise<Enable2FAResponse> => {
    const response = await api.post('/dashboard/auth/2fa/enable', data)
    return response.data
  },

  // Verify 2FA code during login
  verify2FA: async (
    data: Verify2FARequest
  ): Promise<DashboardLoginResponse> => {
    const response = await axios.post(
      `${API_BASE_URL}/dashboard/auth/2fa/verify`,
      data
    )
    return response.data
  },

  // Disable 2FA
  disable2FA: async (data: Disable2FARequest): Promise<{ message: string }> => {
    const response = await api.post('/dashboard/auth/2fa/disable', data)
    return response.data
  },

  // Request password reset
  requestPasswordReset: async (email: string): Promise<{ message: string }> => {
    const response = await axios.post(
      `${API_BASE_URL}/dashboard/auth/password/reset`,
      { email }
    )
    return response.data
  },

  // Verify password reset token
  verifyResetToken: async (
    token: string
  ): Promise<{ valid: boolean; message?: string }> => {
    try {
      const response = await axios.post<{ valid: boolean; message: string }>(
        `${API_BASE_URL}/dashboard/auth/password/reset/verify`,
        { token }
      )
      return response.data
    } catch {
      return { valid: false, message: 'Invalid or expired token' }
    }
  },

  // Reset password with token
  resetPassword: async (
    token: string,
    newPassword: string
  ): Promise<{ message: string }> => {
    const response = await axios.post(
      `${API_BASE_URL}/dashboard/auth/password/reset/confirm`,
      {
        token,
        new_password: newPassword,
      }
    )
    return response.data
  },

  // Get SSO providers available for dashboard login
  getSSOProviders: async (): Promise<{
    providers: SSOProvider[]
    password_login_disabled: boolean
  }> => {
    const response = await axios.get(
      `${API_BASE_URL}/dashboard/auth/sso/providers`
    )
    return response.data
  },
}

// SSO Provider type for dashboard login
export interface SSOProvider {
  id: string
  name: string
  type: 'oauth' | 'saml'
  provider?: string // For OAuth: google, github, etc.
}

// OAuth Provider Management Types
export interface OAuthProviderConfig {
  id: string
  provider_name: string
  display_name: string
  enabled: boolean
  client_id: string
  client_secret?: string
  has_secret: boolean
  redirect_url: string
  scopes: string[]
  is_custom: boolean
  authorization_url?: string
  token_url?: string
  user_info_url?: string
  allow_dashboard_login: boolean
  allow_app_login: boolean
  required_claims?: Record<string, string[]>
  denied_claims?: Record<string, string[]>
  source?: 'database' | 'config'
  created_at: string
  updated_at: string
}

export interface CreateOAuthProviderRequest {
  provider_name: string
  display_name: string
  enabled: boolean
  client_id: string
  client_secret: string
  redirect_url: string
  scopes: string[]
  is_custom: boolean
  authorization_url?: string
  token_url?: string
  user_info_url?: string
  allow_dashboard_login?: boolean
  allow_app_login?: boolean
}

export interface UpdateOAuthProviderRequest {
  display_name?: string
  enabled?: boolean
  client_id?: string
  client_secret?: string
  redirect_url?: string
  scopes?: string[]
  authorization_url?: string
  token_url?: string
  user_info_url?: string
  allow_dashboard_login?: boolean
  allow_app_login?: boolean
}

export interface AuthSettings {
  enable_signup: boolean
  require_email_verification: boolean
  enable_magic_link: boolean
  password_min_length: number
  password_require_uppercase: boolean
  password_require_lowercase: boolean
  password_require_number: boolean
  password_require_special: boolean
  session_timeout_minutes: number
  max_sessions_per_user: number
  disable_dashboard_password_login: boolean
  disable_app_password_login: boolean
}

// OAuth Provider Management API
export const oauthProviderApi = {
  // List all OAuth providers
  list: async (): Promise<OAuthProviderConfig[]> => {
    const response = await api.get<OAuthProviderConfig[]>(
      '/api/v1/admin/oauth/providers'
    )
    return response.data
  },

  // Get single OAuth provider
  get: async (id: string): Promise<OAuthProviderConfig> => {
    const response = await api.get<OAuthProviderConfig>(
      `/api/v1/admin/oauth/providers/${id}`
    )
    return response.data
  },

  // Create OAuth provider
  create: async (
    data: CreateOAuthProviderRequest
  ): Promise<{
    success: boolean
    id: string
    provider: string
    message: string
  }> => {
    const response = await api.post('/api/v1/admin/oauth/providers', data)
    return response.data
  },

  // Update OAuth provider
  update: async (
    id: string,
    data: UpdateOAuthProviderRequest
  ): Promise<{ success: boolean; message: string }> => {
    const response = await api.put(`/api/v1/admin/oauth/providers/${id}`, data)
    return response.data
  },

  // Delete OAuth provider
  delete: async (
    id: string
  ): Promise<{ success: boolean; message: string }> => {
    const response = await api.delete(`/api/v1/admin/oauth/providers/${id}`)
    return response.data
  },
}

// Auth Settings API
export const authSettingsApi = {
  // Get auth settings
  get: async (): Promise<AuthSettings> => {
    const response = await api.get<AuthSettings>('/api/v1/admin/auth/settings')
    return response.data
  },

  // Update auth settings
  update: async (
    data: AuthSettings
  ): Promise<{ success: boolean; message: string }> => {
    const response = await api.put('/api/v1/admin/auth/settings', data)
    return response.data
  },
}

// SAML Provider Management Types
export interface SAMLProviderConfig {
  id: string
  name: string
  display_name: string
  enabled: boolean
  entity_id: string
  acs_url: string
  idp_metadata_url?: string
  idp_metadata_xml?: string
  idp_entity_id?: string
  idp_sso_url?: string
  attribute_mapping: Record<string, string>
  auto_create_users: boolean
  default_role: string
  allow_dashboard_login: boolean
  allow_app_login: boolean
  allow_idp_initiated: boolean
  allowed_redirect_hosts: string[]
  required_groups?: string[]
  required_groups_all?: string[]
  denied_groups?: string[]
  group_attribute?: string
  source: 'database' | 'config'
  created_at: string
  updated_at: string
}

export interface CreateSAMLProviderRequest {
  name: string
  display_name?: string
  enabled: boolean
  idp_metadata_url?: string
  idp_metadata_xml?: string
  attribute_mapping?: Record<string, string>
  auto_create_users?: boolean
  default_role?: string
  allow_dashboard_login?: boolean
  allow_app_login?: boolean
  allow_idp_initiated?: boolean
  allowed_redirect_hosts?: string[]
  required_groups?: string[]
  required_groups_all?: string[]
  denied_groups?: string[]
  group_attribute?: string
}

export interface UpdateSAMLProviderRequest {
  display_name?: string
  enabled?: boolean
  idp_metadata_url?: string
  idp_metadata_xml?: string
  attribute_mapping?: Record<string, string>
  auto_create_users?: boolean
  default_role?: string
  allow_dashboard_login?: boolean
  allow_app_login?: boolean
  allow_idp_initiated?: boolean
  allowed_redirect_hosts?: string[]
  required_groups?: string[]
  required_groups_all?: string[]
  denied_groups?: string[]
  group_attribute?: string
}

export interface ValidateMetadataResponse {
  valid: boolean
  entity_id?: string
  sso_url?: string
  slo_url?: string
  certificate?: string
  error?: string
}

// SAML Provider Management API
export const samlProviderApi = {
  // List all SAML providers
  list: async (): Promise<SAMLProviderConfig[]> => {
    const response = await api.get<SAMLProviderConfig[]>(
      '/api/v1/admin/saml/providers'
    )
    return response.data
  },

  // Get single SAML provider
  get: async (id: string): Promise<SAMLProviderConfig> => {
    const response = await api.get<SAMLProviderConfig>(
      `/api/v1/admin/saml/providers/${id}`
    )
    return response.data
  },

  // Create SAML provider
  create: async (
    data: CreateSAMLProviderRequest
  ): Promise<{
    success: boolean
    id: string
    provider: string
    entity_id: string
    acs_url: string
    message: string
  }> => {
    const response = await api.post('/api/v1/admin/saml/providers', data)
    return response.data
  },

  // Update SAML provider
  update: async (
    id: string,
    data: UpdateSAMLProviderRequest
  ): Promise<{ success: boolean; message: string }> => {
    const response = await api.put(`/api/v1/admin/saml/providers/${id}`, data)
    return response.data
  },

  // Delete SAML provider
  delete: async (
    id: string
  ): Promise<{ success: boolean; message: string }> => {
    const response = await api.delete(`/api/v1/admin/saml/providers/${id}`)
    return response.data
  },

  // Validate metadata URL or XML
  validateMetadata: async (
    metadataUrl?: string,
    metadataXml?: string
  ): Promise<ValidateMetadataResponse> => {
    const response = await api.post<ValidateMetadataResponse>(
      '/api/v1/admin/saml/validate-metadata',
      {
        metadata_url: metadataUrl,
        metadata_xml: metadataXml,
      }
    )
    return response.data
  },

  // Upload metadata XML file
  uploadMetadata: async (
    file: File
  ): Promise<ValidateMetadataResponse & { metadata?: string }> => {
    const formData = new FormData()
    formData.append('metadata', file)
    const response = await api.post(
      '/api/v1/admin/saml/upload-metadata',
      formData,
      {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      }
    )
    return response.data
  },
}

// AI Providers API
export interface AIProvider {
  id: string
  name: string
  display_name: string
  provider_type: 'openai' | 'azure' | 'ollama'
  is_default: boolean
  config: Record<string, string>
  enabled: boolean
  from_config: boolean // True if configured via environment/YAML (read-only)
  created_at: string
  updated_at: string
  created_by?: string
}

export interface UpdateAIProviderRequest {
  display_name?: string
  config?: Record<string, string>
  enabled?: boolean
}

export const aiProvidersApi = {
  list: async (): Promise<AIProvider[]> => {
    const response = await api.get<{ providers: AIProvider[] }>(
      '/api/v1/admin/ai/providers'
    )
    return response.data.providers
  },
}

// AI Chatbots API
export interface AIChatbotSummary {
  id: string
  name: string
  namespace: string
  description?: string
  model?: string
  enabled: boolean
  is_public: boolean
  allowed_tables: string[]
  allowed_operations: string[]
  allowed_schemas: string[]
  version: number
  source: string
  created_at: string
  updated_at: string
}

export interface AIChatbot extends AIChatbotSummary {
  code: string
  original_code?: string
  max_tokens: number
  temperature: number
  provider_id?: string
  persist_conversations: boolean
  conversation_ttl_hours: number
  max_conversation_turns: number
  rate_limit_per_minute: number
  daily_request_limit: number
  daily_token_budget: number
  allow_unauthenticated: boolean
}

export const chatbotsApi = {
  // List all chatbots
  list: async (namespace?: string): Promise<AIChatbotSummary[]> => {
    const params = namespace ? `?namespace=${namespace}` : ''
    const response = await api.get<{
      chatbots: AIChatbotSummary[]
      count: number
    }>(`/api/v1/admin/ai/chatbots${params}`)
    return response.data.chatbots || []
  },

  // Get chatbot details
  get: async (id: string): Promise<AIChatbot> => {
    const response = await api.get<AIChatbot>(`/api/v1/admin/ai/chatbots/${id}`)
    return response.data
  },

  // Toggle chatbot enabled status
  toggle: async (id: string, enabled: boolean): Promise<AIChatbot> => {
    const response = await api.put<AIChatbot>(
      `/api/v1/admin/ai/chatbots/${id}/toggle`,
      { enabled }
    )
    return response.data
  },

  // Update chatbot configuration
  update: async (id: string, data: Partial<AIChatbot>): Promise<AIChatbot> => {
    const response = await api.put<AIChatbot>(
      `/api/v1/admin/ai/chatbots/${id}`,
      data
    )
    return response.data
  },

  // Delete chatbot
  delete: async (id: string): Promise<void> => {
    await api.delete(`/api/v1/admin/ai/chatbots/${id}`)
  },

  // Sync chatbots from filesystem
  sync: async (): Promise<{
    summary: {
      created: number
      updated: number
      deleted: number
      errors: number
    }
  }> => {
    const response = await api.post<{
      summary: {
        created: number
        updated: number
        deleted: number
        errors: number
      }
    }>('/api/v1/admin/ai/chatbots/sync', {})
    return response.data
  },
}

// AI Metrics API
export interface AIMetrics {
  total_requests: number
  total_tokens: number
  total_prompt_tokens: number
  total_completion_tokens: number
  active_conversations: number
  total_conversations: number
  chatbot_stats: Array<{
    chatbot_id: string
    chatbot_name: string
    requests: number
    tokens: number
    error_count: number
  }>
  provider_stats: Array<{
    provider_id: string
    provider_name: string
    requests: number
    avg_latency_ms: number
  }>
  error_rate: number
  avg_response_time_ms: number
}

export const aiMetricsApi = {
  getMetrics: async (): Promise<AIMetrics> => {
    const response = await api.get<AIMetrics>('/api/v1/admin/ai/metrics')
    return response.data
  },
}

// AI Conversations API
export interface ConversationSummary {
  id: string
  chatbot_id: string
  chatbot_name: string
  user_id?: string
  user_email?: string
  session_id?: string
  title?: string
  status: string
  turn_count: number
  total_prompt_tokens: number
  total_completion_tokens: number
  created_at: string
  updated_at: string
  last_message_at: string
}

export interface MessageDetail {
  id: string
  conversation_id: string
  role: string
  content: string
  tool_call_id?: string
  tool_name?: string
  executed_sql?: string
  sql_result_summary?: string
  sql_row_count?: number
  sql_error?: string
  sql_duration_ms?: number
  prompt_tokens?: number
  completion_tokens?: number
  created_at: string
  sequence_number: number
}

export const conversationsApi = {
  list: async (params?: {
    chatbot_id?: string
    user_id?: string
    status?: string
    limit?: number
    offset?: number
  }): Promise<{
    conversations: ConversationSummary[]
    total: number
    total_count: number
  }> => {
    const queryParams = new URLSearchParams()
    if (params?.chatbot_id) queryParams.append('chatbot_id', params.chatbot_id)
    if (params?.user_id) queryParams.append('user_id', params.user_id)
    if (params?.status) queryParams.append('status', params.status)
    if (params?.limit) queryParams.append('limit', params.limit.toString())
    if (params?.offset) queryParams.append('offset', params.offset.toString())

    const response = await api.get<{
      conversations: ConversationSummary[]
      total: number
      total_count: number
    }>(`/api/v1/admin/ai/conversations?${queryParams.toString()}`)
    return response.data
  },

  getMessages: async (
    conversationId: string
  ): Promise<{ messages: MessageDetail[]; total: number }> => {
    const response = await api.get<{
      messages: MessageDetail[]
      total: number
    }>(`/api/v1/admin/ai/conversations/${conversationId}/messages`)
    return response.data
  },
}

// AI Audit Log API
export interface AuditLogEntry {
  id: string
  chatbot_id?: string
  chatbot_name?: string
  conversation_id?: string
  message_id?: string
  user_id?: string
  user_email?: string
  generated_sql: string
  sanitized_sql?: string
  executed: boolean
  validation_passed?: boolean
  validation_errors?: string[]
  success?: boolean
  error_message?: string
  rows_returned?: number
  execution_duration_ms?: number
  tables_accessed?: string[]
  operations_used?: string[]
  ip_address?: string
  user_agent?: string
  created_at: string
}

export const auditLogApi = {
  list: async (params?: {
    chatbot_id?: string
    user_id?: string
    success?: boolean
    limit?: number
    offset?: number
  }): Promise<{
    entries: AuditLogEntry[]
    total: number
    total_count: number
  }> => {
    const queryParams = new URLSearchParams()
    if (params?.chatbot_id) queryParams.append('chatbot_id', params.chatbot_id)
    if (params?.user_id) queryParams.append('user_id', params.user_id)
    if (params?.success !== undefined)
      queryParams.append('success', params.success.toString())
    if (params?.limit) queryParams.append('limit', params.limit.toString())
    if (params?.offset) queryParams.append('offset', params.offset.toString())

    const response = await api.get<{
      entries: AuditLogEntry[]
      total: number
      total_count: number
    }>(`/api/v1/admin/ai/audit?${queryParams.toString()}`)
    return response.data
  },
}

// RPC (Remote Procedure Call) API Types
export interface RPCProcedure {
  id: string
  name: string
  namespace: string
  description?: string
  sql_query: string
  original_code?: string
  input_schema?: Record<string, string>
  output_schema?: Record<string, string>
  allowed_tables: string[]
  allowed_schemas: string[]
  max_execution_time_seconds: number
  require_role?: string
  is_public: boolean
  schedule?: string
  enabled: boolean
  version: number
  source: string
  created_by?: string
  created_at: string
  updated_at: string
}

export type RPCExecutionStatus =
  | 'pending'
  | 'running'
  | 'completed'
  | 'failed'
  | 'cancelled'
  | 'timeout'

export interface RPCExecution {
  id: string
  procedure_id?: string
  procedure_name: string
  namespace: string
  status: RPCExecutionStatus
  input_params?: Record<string, unknown>
  result?: unknown
  error_message?: string
  rows_returned?: number
  duration_ms?: number
  user_id?: string
  user_role?: string
  user_email?: string
  is_async: boolean
  created_at: string
  started_at?: string
  completed_at?: string
}

// Note: RPCExecutionLog is now stored in the central logging schema (logging.entries)

export interface RPCSyncResult {
  message: string
  namespace: string
  summary: {
    created: number
    updated: number
    deleted: number
    unchanged: number
    errors: number
  }
  details: {
    created: string[]
    updated: string[]
    deleted: string[]
    unchanged: string[]
  }
  errors: Array<{ procedure: string; error: string }>
  dry_run: boolean
}

export interface UpdateRPCProcedureRequest {
  description?: string
  enabled?: boolean
  is_public?: boolean
  require_role?: string
  max_execution_time_seconds?: number
  allowed_tables?: string[]
  allowed_schemas?: string[]
  schedule?: string
}

// RPC API
export const rpcApi = {
  // List all namespaces
  listNamespaces: async (): Promise<string[]> => {
    const response = await api.get<{ namespaces: string[] }>(
      '/api/v1/admin/rpc/namespaces'
    )
    return response.data.namespaces || ['default']
  },

  // List all procedures (optionally filtered by namespace)
  listProcedures: async (namespace?: string): Promise<RPCProcedure[]> => {
    const params = namespace ? `?namespace=${namespace}` : ''
    const response = await api.get<{
      procedures: RPCProcedure[]
      count: number
    }>(`/api/v1/admin/rpc/procedures${params}`)
    return response.data.procedures || []
  },

  // Get procedure details
  getProcedure: async (
    namespace: string,
    name: string
  ): Promise<RPCProcedure> => {
    const response = await api.get<RPCProcedure>(
      `/api/v1/admin/rpc/procedures/${namespace}/${name}`
    )
    return response.data
  },

  // Update procedure
  updateProcedure: async (
    namespace: string,
    name: string,
    data: UpdateRPCProcedureRequest
  ): Promise<RPCProcedure> => {
    const response = await api.put<RPCProcedure>(
      `/api/v1/admin/rpc/procedures/${namespace}/${name}`,
      data
    )
    return response.data
  },

  // Delete procedure
  deleteProcedure: async (namespace: string, name: string): Promise<void> => {
    await api.delete(`/api/v1/admin/rpc/procedures/${namespace}/${name}`)
  },

  // Sync procedures from filesystem
  sync: async (namespace: string): Promise<RPCSyncResult> => {
    const response = await api.post<RPCSyncResult>('/api/v1/admin/rpc/sync', {
      namespace,
    })
    return response.data
  },

  // List executions with filters
  listExecutions: async (filters?: {
    namespace?: string
    procedure?: string
    status?: RPCExecutionStatus
    limit?: number
    offset?: number
  }): Promise<{ executions: RPCExecution[]; total: number }> => {
    const params = new URLSearchParams()
    if (filters?.namespace) params.set('namespace', filters.namespace)
    if (filters?.procedure) params.set('procedure', filters.procedure)
    if (filters?.status) params.set('status', filters.status)
    if (filters?.limit) params.set('limit', filters.limit.toString())
    if (filters?.offset) params.set('offset', filters.offset.toString())

    const queryString = params.toString()
    const response = await api.get<{
      executions: RPCExecution[]
      count: number
    }>(`/api/v1/admin/rpc/executions${queryString ? `?${queryString}` : ''}`)
    return {
      executions: response.data.executions || [],
      total: response.data.count || 0,
    }
  },

  // Get execution details
  getExecution: async (executionId: string): Promise<RPCExecution> => {
    const response = await api.get<RPCExecution>(
      `/api/v1/admin/rpc/executions/${executionId}`
    )
    return response.data
  },

  // Note: Execution logs are now stored in the central logging schema (logging.entries)
  // Use the logsApi to query execution logs

  // Cancel execution
  cancelExecution: async (executionId: string): Promise<void> => {
    await api.post(`/api/v1/admin/rpc/executions/${executionId}/cancel`)
  },
}

// ============================================================================
// Knowledge Base Types and API
// ============================================================================

export interface KnowledgeBaseSummary {
  id: string
  name: string
  namespace: string
  description: string
  enabled: boolean
  document_count: number
  total_chunks: number
  embedding_model: string
  created_at: string
  updated_at: string
}

export interface KnowledgeBase extends KnowledgeBaseSummary {
  embedding_dimensions: number
  chunk_size: number
  chunk_overlap: number
  chunk_strategy: string
  source: string
  created_by?: string
}

export interface CreateKnowledgeBaseRequest {
  name: string
  namespace?: string
  description?: string
  embedding_model?: string
  embedding_dimensions?: number
  chunk_size?: number
  chunk_overlap?: number
  chunk_strategy?: string
}

export interface UpdateKnowledgeBaseRequest {
  name?: string
  description?: string
  embedding_model?: string
  embedding_dimensions?: number
  chunk_size?: number
  chunk_overlap?: number
  chunk_strategy?: string
  enabled?: boolean
}

export type DocumentStatus = 'pending' | 'processing' | 'indexed' | 'failed'

export interface KnowledgeBaseDocument {
  id: string
  knowledge_base_id: string
  title: string
  source_url?: string
  source_type?: string
  mime_type: string
  content_hash: string
  chunk_count: number
  status: DocumentStatus
  error_message?: string
  metadata?: Record<string, string>
  tags?: string[]
  created_at: string
  updated_at: string
}

export interface AddDocumentRequest {
  title?: string
  content: string
  source?: string
  mime_type?: string
  metadata?: Record<string, string>
  tags?: string[]
}

export interface AddDocumentResponse {
  document_id: string
  status: string
  message: string
}

export interface ChatbotKnowledgeBaseLink {
  id: string
  chatbot_id: string
  knowledge_base_id: string
  enabled: boolean
  max_chunks: number
  similarity_threshold: number
  priority: number
  created_at: string
}

export interface SearchResult {
  chunk_id: string
  document_id: string
  document_title: string
  knowledge_base_name?: string
  content: string
  similarity: number
}

export interface DebugSearchResult {
  query: string
  query_embedding_preview: number[]
  query_embedding_dims: number
  stored_embedding_preview?: number[]
  raw_similarities: number[]
  embedding_model: string
  kb_embedding_model: string
  chunks_found: number
  top_chunk_content_preview?: string
  // Chunk statistics
  total_chunks: number
  chunks_with_embedding: number
  chunks_without_embedding: number
  error_message?: string
}

export const knowledgeBasesApi = {
  // List all knowledge bases
  list: async (): Promise<KnowledgeBaseSummary[]> => {
    const response = await api.get<{
      knowledge_bases: KnowledgeBaseSummary[]
      count: number
    }>('/api/v1/admin/ai/knowledge-bases')
    return response.data.knowledge_bases || []
  },

  // Get knowledge base details
  get: async (id: string): Promise<KnowledgeBase> => {
    const response = await api.get<KnowledgeBase>(
      `/api/v1/admin/ai/knowledge-bases/${id}`
    )
    return response.data
  },

  // Create a new knowledge base
  create: async (data: CreateKnowledgeBaseRequest): Promise<KnowledgeBase> => {
    const response = await api.post<KnowledgeBase>(
      '/api/v1/admin/ai/knowledge-bases',
      data
    )
    return response.data
  },

  // Update knowledge base
  update: async (
    id: string,
    data: UpdateKnowledgeBaseRequest
  ): Promise<KnowledgeBase> => {
    const response = await api.put<KnowledgeBase>(
      `/api/v1/admin/ai/knowledge-bases/${id}`,
      data
    )
    return response.data
  },

  // Delete knowledge base
  delete: async (id: string): Promise<void> => {
    await api.delete(`/api/v1/admin/ai/knowledge-bases/${id}`)
  },

  // List documents in a knowledge base
  listDocuments: async (kbId: string): Promise<KnowledgeBaseDocument[]> => {
    const response = await api.get<{
      documents: KnowledgeBaseDocument[]
      count: number
    }>(`/api/v1/admin/ai/knowledge-bases/${kbId}/documents`)
    return response.data.documents || []
  },

  // Get document details
  getDocument: async (
    kbId: string,
    docId: string
  ): Promise<KnowledgeBaseDocument> => {
    const response = await api.get<KnowledgeBaseDocument>(
      `/api/v1/admin/ai/knowledge-bases/${kbId}/documents/${docId}`
    )
    return response.data
  },

  // Add a document
  addDocument: async (
    kbId: string,
    data: AddDocumentRequest
  ): Promise<AddDocumentResponse> => {
    const response = await api.post<AddDocumentResponse>(
      `/api/v1/admin/ai/knowledge-bases/${kbId}/documents`,
      data
    )
    return response.data
  },

  // Delete document
  deleteDocument: async (kbId: string, docId: string): Promise<void> => {
    await api.delete(
      `/api/v1/admin/ai/knowledge-bases/${kbId}/documents/${docId}`
    )
  },

  // Update document metadata and tags
  updateDocument: async (
    kbId: string,
    docId: string,
    data: {
      title?: string
      metadata?: Record<string, string>
      tags?: string[]
    }
  ): Promise<KnowledgeBaseDocument> => {
    const response = await api.patch<KnowledgeBaseDocument>(
      `/api/v1/admin/ai/knowledge-bases/${kbId}/documents/${docId}`,
      data
    )
    return response.data
  },

  // Get knowledge base capabilities (OCR status, supported file types)
  getCapabilities: async (): Promise<{
    ocr_enabled: boolean
    ocr_available: boolean
    ocr_languages: string[]
    supported_file_types: string[]
  }> => {
    const response = await api.get<{
      ocr_enabled: boolean
      ocr_available: boolean
      ocr_languages: string[]
      supported_file_types: string[]
    }>('/api/v1/admin/ai/knowledge-bases/capabilities')
    return response.data
  },

  // Upload document file
  uploadDocument: async (
    kbId: string,
    file: File,
    title?: string
  ): Promise<{
    document_id: string
    status: string
    message: string
    filename: string
    extracted_length: number
    mime_type: string
  }> => {
    const formData = new FormData()
    formData.append('file', file)
    if (title) {
      formData.append('title', title)
    }
    const response = await api.post<{
      document_id: string
      status: string
      message: string
      filename: string
      extracted_length: number
      mime_type: string
    }>(`/api/v1/admin/ai/knowledge-bases/${kbId}/documents/upload`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    })
    return response.data
  },

  // Search knowledge base
  search: async (
    kbId: string,
    query: string,
    options?: {
      max_chunks?: number
      threshold?: number
      mode?: 'semantic' | 'keyword' | 'hybrid'
      semantic_weight?: number
    }
  ): Promise<{
    results: SearchResult[]
    count: number
    query: string
    mode: string
  }> => {
    const response = await api.post<{
      results: SearchResult[]
      count: number
      query: string
      mode: string
    }>(`/api/v1/admin/ai/knowledge-bases/${kbId}/search`, {
      query,
      max_chunks: options?.max_chunks,
      threshold: options?.threshold,
      mode: options?.mode,
      semantic_weight: options?.semantic_weight,
    })
    return response.data
  },

  // Debug search - returns detailed diagnostic information
  debugSearch: async (
    kbId: string,
    query: string
  ): Promise<DebugSearchResult> => {
    const response = await api.post<DebugSearchResult>(
      `/api/v1/admin/ai/knowledge-bases/${kbId}/debug-search`,
      { query }
    )
    return response.data
  },

  // List chatbot knowledge base links
  listChatbotLinks: async (
    chatbotId: string
  ): Promise<ChatbotKnowledgeBaseLink[]> => {
    const response = await api.get<{
      knowledge_bases: ChatbotKnowledgeBaseLink[]
      count: number
    }>(`/api/v1/admin/ai/chatbots/${chatbotId}/knowledge-bases`)
    return response.data.knowledge_bases || []
  },

  // Link knowledge base to chatbot
  linkToChatbot: async (
    chatbotId: string,
    kbId: string,
    options?: {
      priority?: number
      max_chunks?: number
      similarity_threshold?: number
    }
  ): Promise<ChatbotKnowledgeBaseLink> => {
    const response = await api.post<ChatbotKnowledgeBaseLink>(
      `/api/v1/admin/ai/chatbots/${chatbotId}/knowledge-bases`,
      {
        knowledge_base_id: kbId,
        ...options,
      }
    )
    return response.data
  },

  // Update chatbot knowledge base link
  updateChatbotLink: async (
    chatbotId: string,
    kbId: string,
    data: {
      priority?: number
      max_chunks?: number
      similarity_threshold?: number
      enabled?: boolean
    }
  ): Promise<ChatbotKnowledgeBaseLink> => {
    const response = await api.put<ChatbotKnowledgeBaseLink>(
      `/api/v1/admin/ai/chatbots/${chatbotId}/knowledge-bases/${kbId}`,
      data
    )
    return response.data
  },

  // Unlink knowledge base from chatbot
  unlinkFromChatbot: async (chatbotId: string, kbId: string): Promise<void> => {
    await api.delete(
      `/api/v1/admin/ai/chatbots/${chatbotId}/knowledge-bases/${kbId}`
    )
  },
}

// =============================================================================
// Logs API
// =============================================================================

export interface LogEntryAPI {
  id: string
  timestamp: string
  category: string
  level: string
  message: string
  custom_category?: string
  request_id?: string
  trace_id?: string
  component?: string
  user_id?: string
  ip_address?: string
  fields?: Record<string, unknown>
  execution_id?: string
  execution_type?: string
  line_number?: number
}

export interface LogQueryOptionsAPI {
  category?: string
  custom_category?: string
  levels?: string[]
  component?: string
  request_id?: string
  trace_id?: string
  user_id?: string
  execution_id?: string
  search?: string
  start_time?: string
  end_time?: string
  limit?: number
  offset?: number
  sort_asc?: boolean
  hide_static_assets?: boolean
}

export interface LogQueryResultAPI {
  entries: LogEntryAPI[]
  total_count: number
  has_more: boolean
}

export interface LogStatsAPI {
  total_entries: number
  entries_by_category: Record<string, number>
  entries_by_level: Record<string, number>
  oldest_entry?: string
  newest_entry?: string
}

export const logsApi = {
  // Query logs with filters
  query: async (options: LogQueryOptionsAPI): Promise<LogQueryResultAPI> => {
    const params = new URLSearchParams()
    if (options.category) params.set('category', options.category)
    if (options.custom_category)
      params.set('custom_category', options.custom_category)
    if (options.levels?.length) params.set('level', options.levels.join(','))
    if (options.component) params.set('component', options.component)
    if (options.request_id) params.set('request_id', options.request_id)
    if (options.trace_id) params.set('trace_id', options.trace_id)
    if (options.user_id) params.set('user_id', options.user_id)
    if (options.execution_id) params.set('execution_id', options.execution_id)
    if (options.search) params.set('search', options.search)
    if (options.start_time) params.set('start_time', options.start_time)
    if (options.end_time) params.set('end_time', options.end_time)
    if (options.limit) params.set('limit', options.limit.toString())
    if (options.offset) params.set('offset', options.offset.toString())
    if (options.sort_asc) params.set('sort_asc', 'true')
    if (options.hide_static_assets) params.set('hide_static_assets', 'true')

    const response = await api.get<LogQueryResultAPI>(
      `/api/v1/admin/logs?${params.toString()}`
    )
    return response.data
  },

  // Get log statistics
  getStats: async (): Promise<LogStatsAPI> => {
    const response = await api.get<LogStatsAPI>('/api/v1/admin/logs/stats')
    return response.data
  },

  // Get execution logs
  getExecutionLogs: async (
    executionId: string,
    afterLine?: number
  ): Promise<{ entries: LogEntryAPI[]; count: number }> => {
    const params = afterLine ? `?after_line=${afterLine}` : ''
    const response = await api.get<{ entries: LogEntryAPI[]; count: number }>(
      `/api/v1/admin/logs/executions/${executionId}${params}`
    )
    return response.data
  },
}

// Secrets API Types
export interface Secret {
  id: string
  name: string
  scope: 'global' | 'namespace'
  namespace?: string
  description?: string
  version: number
  expires_at?: string
  is_expired?: boolean
  created_at: string
  updated_at: string
  created_by?: string
  updated_by?: string
}

export interface SecretVersion {
  id: string
  secret_id: string
  version: number
  created_at: string
  created_by?: string
}

export interface CreateSecretRequest {
  name: string
  value: string
  scope: 'global' | 'namespace'
  namespace?: string
  description?: string
  expires_at?: string
}

export interface UpdateSecretRequest {
  value?: string
  description?: string
  expires_at?: string
}

export interface SecretsStats {
  total: number
  expiring_soon: number
  expired: number
}

// Secrets API
export const secretsApi = {
  // List all secrets (metadata only, never includes values)
  list: async (scope?: string, namespace?: string): Promise<Secret[]> => {
    const params = new URLSearchParams()
    if (scope) params.set('scope', scope)
    if (namespace) params.set('namespace', namespace)
    const queryString = params.toString()
    const response = await api.get<Secret[]>(
      `/api/v1/secrets${queryString ? `?${queryString}` : ''}`
    )
    return response.data
  },

  // Get a secret by ID (metadata only)
  get: async (id: string): Promise<Secret> => {
    const response = await api.get<Secret>(`/api/v1/secrets/${id}`)
    return response.data
  },

  // Create a new secret
  create: async (request: CreateSecretRequest): Promise<Secret> => {
    const response = await api.post<Secret>('/api/v1/secrets', request)
    return response.data
  },

  // Update a secret
  update: async (id: string, request: UpdateSecretRequest): Promise<Secret> => {
    const response = await api.put<Secret>(`/api/v1/secrets/${id}`, request)
    return response.data
  },

  // Delete a secret
  delete: async (id: string): Promise<{ message: string }> => {
    const response = await api.delete<{ message: string }>(
      `/api/v1/secrets/${id}`
    )
    return response.data
  },

  // Get version history for a secret
  getVersions: async (id: string): Promise<SecretVersion[]> => {
    const response = await api.get<SecretVersion[]>(
      `/api/v1/secrets/${id}/versions`
    )
    return response.data
  },

  // Rollback to a previous version
  rollback: async (
    id: string,
    version: number
  ): Promise<{ message: string }> => {
    const response = await api.post<{ message: string }>(
      `/api/v1/secrets/${id}/rollback/${version}`
    )
    return response.data
  },

  // Get secrets statistics
  getStats: async (): Promise<SecretsStats> => {
    const response = await api.get<SecretsStats>('/api/v1/secrets/stats')
    return response.data
  },
}

// Service Keys API Types
export interface ServiceKey {
  id: string
  name: string
  description?: string
  key_prefix: string
  scopes: string[]
  enabled: boolean
  rate_limit_per_minute?: number
  rate_limit_per_hour?: number
  created_by?: string
  created_at: string
  last_used_at?: string
  expires_at?: string
  // Revocation fields
  revoked_at?: string
  revoked_by?: string
  revocation_reason?: string
  // Deprecation fields
  deprecated_at?: string
  grace_period_ends_at?: string
  replaced_by?: string
}

export interface ServiceKeyRevocation {
  id: string
  service_key_id: string
  revocation_type: 'emergency' | 'rotation' | 'expiration' | 'deprecation'
  reason: string
  revoked_by: string
  created_at: string
}

export interface RevokeServiceKeyRequest {
  reason: string
}

export interface DeprecateServiceKeyRequest {
  grace_period: string
  reason?: string
}

export interface RotateServiceKeyRequest {
  grace_period: string
}

export interface RotateServiceKeyResponse extends ServiceKeyWithPlaintext {
  grace_period_ends_at: string
}

export interface ServiceKeyWithPlaintext extends ServiceKey {
  key: string // Only returned on creation
}

export interface CreateServiceKeyRequest {
  name: string
  description?: string
  scopes?: string[]
  rate_limit_per_minute?: number
  rate_limit_per_hour?: number
  expires_at?: string
}

export interface UpdateServiceKeyRequest {
  name?: string
  description?: string
  scopes?: string[]
  enabled?: boolean
  rate_limit_per_minute?: number
  rate_limit_per_hour?: number
}

// Service Keys API
export const serviceKeysApi = {
  // List all service keys
  list: async (): Promise<ServiceKey[]> => {
    const response = await api.get<ServiceKey[]>('/api/v1/admin/service-keys')
    return response.data
  },

  // Get a specific service key
  get: async (id: string): Promise<ServiceKey> => {
    const response = await api.get<ServiceKey>(
      `/api/v1/admin/service-keys/${id}`
    )
    return response.data
  },

  // Create a new service key
  create: async (
    request: CreateServiceKeyRequest
  ): Promise<ServiceKeyWithPlaintext> => {
    const response = await api.post<ServiceKeyWithPlaintext>(
      '/api/v1/admin/service-keys',
      request
    )
    return response.data
  },

  // Update a service key
  update: async (
    id: string,
    request: UpdateServiceKeyRequest
  ): Promise<{ success: boolean; message: string }> => {
    const response = await api.patch<{ success: boolean; message: string }>(
      `/api/v1/admin/service-keys/${id}`,
      request
    )
    return response.data
  },

  // Delete a service key
  delete: async (id: string): Promise<void> => {
    await api.delete(`/api/v1/admin/service-keys/${id}`)
  },

  // Disable a service key
  disable: async (
    id: string
  ): Promise<{ success: boolean; message: string }> => {
    const response = await api.post<{ success: boolean; message: string }>(
      `/api/v1/admin/service-keys/${id}/disable`
    )
    return response.data
  },

  // Enable a service key
  enable: async (
    id: string
  ): Promise<{ success: boolean; message: string }> => {
    const response = await api.post<{ success: boolean; message: string }>(
      `/api/v1/admin/service-keys/${id}/enable`
    )
    return response.data
  },

  // Revoke a service key (emergency, immediate)
  revoke: async (
    id: string,
    request: RevokeServiceKeyRequest
  ): Promise<{ success: boolean; message: string }> => {
    const response = await api.post<{ success: boolean; message: string }>(
      `/api/v1/admin/service-keys/${id}/revoke`,
      request
    )
    return response.data
  },

  // Deprecate a service key (with grace period)
  deprecate: async (
    id: string,
    request: DeprecateServiceKeyRequest
  ): Promise<ServiceKey> => {
    const response = await api.post<ServiceKey>(
      `/api/v1/admin/service-keys/${id}/deprecate`,
      request
    )
    return response.data
  },

  // Rotate a service key (create replacement)
  rotate: async (
    id: string,
    request: RotateServiceKeyRequest
  ): Promise<RotateServiceKeyResponse> => {
    const response = await api.post<RotateServiceKeyResponse>(
      `/api/v1/admin/service-keys/${id}/rotate`,
      request
    )
    return response.data
  },

  // Get revocation history for a service key
  revocations: async (id: string): Promise<ServiceKeyRevocation[]> => {
    const response = await api.get<ServiceKeyRevocation[]>(
      `/api/v1/admin/service-keys/${id}/revocations`
    )
    return response.data
  },
}

// Captcha Settings Types
export interface CaptchaSettingsResponse {
  enabled: boolean
  provider: string
  site_key: string
  secret_key_set: boolean
  score_threshold: number
  endpoints: string[]
  cap_server_url: string
  cap_api_key_set: boolean
  _overrides: Record<
    string,
    {
      is_overridden: boolean
      env_var?: string
    }
  >
}

export interface UpdateCaptchaSettingsRequest {
  enabled?: boolean
  provider?: string
  site_key?: string
  secret_key?: string
  score_threshold?: number
  endpoints?: string[]
  cap_server_url?: string
  cap_api_key?: string
}

// Captcha Settings API
export const captchaSettingsApi = {
  // Get current captcha settings
  get: async (): Promise<CaptchaSettingsResponse> => {
    const response = await api.get<CaptchaSettingsResponse>(
      '/api/v1/admin/settings/captcha'
    )
    return response.data
  },

  // Update captcha settings
  update: async (
    request: UpdateCaptchaSettingsRequest
  ): Promise<CaptchaSettingsResponse> => {
    const response = await api.put<CaptchaSettingsResponse>(
      '/api/v1/admin/settings/captcha',
      request
    )
    return response.data
  },
}

// Custom MCP Tools API Types
export interface MCPTool {
  id: string
  name: string
  namespace: string
  description?: string
  code: string
  input_schema?: Record<string, unknown>
  required_scopes: string[]
  timeout_seconds: number
  memory_limit_mb: number
  allow_net: boolean
  allow_env: boolean
  allow_read: boolean
  allow_write: boolean
  enabled: boolean
  created_at: string
  updated_at: string
  created_by?: string
}

export interface MCPResource {
  id: string
  uri: string
  name: string
  namespace: string
  description?: string
  mime_type: string
  code: string
  required_scopes: string[]
  timeout_seconds: number
  memory_limit_mb: number
  allow_net: boolean
  allow_env: boolean
  enabled: boolean
  is_template: boolean
  created_at: string
  updated_at: string
  created_by?: string
}

export interface CreateMCPToolRequest {
  name: string
  namespace?: string
  description?: string
  code: string
  input_schema?: Record<string, unknown>
  required_scopes?: string[]
  timeout_seconds?: number
  memory_limit_mb?: number
  allow_net?: boolean
  allow_env?: boolean
  allow_read?: boolean
  allow_write?: boolean
  enabled?: boolean
}

export interface UpdateMCPToolRequest {
  name?: string
  namespace?: string
  description?: string
  code?: string
  input_schema?: Record<string, unknown>
  required_scopes?: string[]
  timeout_seconds?: number
  memory_limit_mb?: number
  allow_net?: boolean
  allow_env?: boolean
  allow_read?: boolean
  allow_write?: boolean
  enabled?: boolean
}

export interface CreateMCPResourceRequest {
  uri: string
  name: string
  namespace?: string
  description?: string
  mime_type?: string
  code: string
  required_scopes?: string[]
  timeout_seconds?: number
  memory_limit_mb?: number
  allow_net?: boolean
  allow_env?: boolean
  enabled?: boolean
}

export interface UpdateMCPResourceRequest {
  uri?: string
  name?: string
  namespace?: string
  description?: string
  mime_type?: string
  code?: string
  required_scopes?: string[]
  timeout_seconds?: number
  memory_limit_mb?: number
  allow_net?: boolean
  allow_env?: boolean
  enabled?: boolean
}

export interface MCPToolTestResult {
  content: Array<{ type: string; text: string }>
  isError?: boolean
}

export interface MCPResourceTestResult {
  uri: string
  mimeType?: string
  text?: string
  blob?: string
}

// Custom MCP Tools API
export const mcpToolsApi = {
  // List all custom MCP tools
  list: async (namespace?: string): Promise<MCPTool[]> => {
    const params = namespace ? `?namespace=${namespace}` : ''
    const response = await api.get<{ tools: MCPTool[]; count: number }>(
      `/api/v1/mcp/tools${params}`
    )
    return response.data.tools || []
  },

  // Get a single tool by ID
  get: async (id: string): Promise<MCPTool> => {
    const response = await api.get<MCPTool>(`/api/v1/mcp/tools/${id}`)
    return response.data
  },

  // Create a new tool
  create: async (data: CreateMCPToolRequest): Promise<MCPTool> => {
    const response = await api.post<MCPTool>('/api/v1/mcp/tools', data)
    return response.data
  },

  // Update an existing tool
  update: async (id: string, data: UpdateMCPToolRequest): Promise<MCPTool> => {
    const response = await api.put<MCPTool>(`/api/v1/mcp/tools/${id}`, data)
    return response.data
  },

  // Delete a tool
  delete: async (id: string): Promise<void> => {
    await api.delete(`/api/v1/mcp/tools/${id}`)
  },

  // Sync (upsert) a tool
  sync: async (data: CreateMCPToolRequest): Promise<MCPTool> => {
    const response = await api.post<MCPTool>('/api/v1/mcp/tools/sync', {
      ...data,
      upsert: true,
    })
    return response.data
  },

  // Test a tool
  test: async (
    id: string,
    args: Record<string, unknown>
  ): Promise<{ success: boolean; result: MCPToolTestResult }> => {
    const response = await api.post<{
      success: boolean
      result: MCPToolTestResult
    }>(`/api/v1/mcp/tools/${id}/test`, { args })
    return response.data
  },
}

// Custom MCP Resources API
export const mcpResourcesApi = {
  // List all custom MCP resources
  list: async (namespace?: string): Promise<MCPResource[]> => {
    const params = namespace ? `?namespace=${namespace}` : ''
    const response = await api.get<{ resources: MCPResource[]; count: number }>(
      `/api/v1/mcp/resources${params}`
    )
    return response.data.resources || []
  },

  // Get a single resource by ID
  get: async (id: string): Promise<MCPResource> => {
    const response = await api.get<MCPResource>(`/api/v1/mcp/resources/${id}`)
    return response.data
  },

  // Create a new resource
  create: async (data: CreateMCPResourceRequest): Promise<MCPResource> => {
    const response = await api.post<MCPResource>('/api/v1/mcp/resources', data)
    return response.data
  },

  // Update an existing resource
  update: async (
    id: string,
    data: UpdateMCPResourceRequest
  ): Promise<MCPResource> => {
    const response = await api.put<MCPResource>(
      `/api/v1/mcp/resources/${id}`,
      data
    )
    return response.data
  },

  // Delete a resource
  delete: async (id: string): Promise<void> => {
    await api.delete(`/api/v1/mcp/resources/${id}`)
  },

  // Sync (upsert) a resource
  sync: async (data: CreateMCPResourceRequest): Promise<MCPResource> => {
    const response = await api.post<MCPResource>('/api/v1/mcp/resources/sync', {
      ...data,
      upsert: true,
    })
    return response.data
  },

  // Test a resource
  test: async (
    id: string,
    params: Record<string, string>
  ): Promise<{ success: boolean; contents: MCPResourceTestResult[] }> => {
    const response = await api.post<{
      success: boolean
      contents: MCPResourceTestResult[]
    }>(`/api/v1/mcp/resources/${id}/test`, { params })
    return response.data
  },
}
