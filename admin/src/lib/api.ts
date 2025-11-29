import axios, { type AxiosError, type AxiosInstance } from 'axios'
import {
  getAccessToken,
  getRefreshToken,
  setTokens,
  clearTokens,
  type AdminUser,
} from './auth'

// Base URL for the API - can be overridden with environment variable
// Use empty string (relative URLs) to work with both dev server proxy and production
const API_BASE_URL = import.meta.env.VITE_API_URL || ''

// Helper to get impersonation token if active
const getImpersonationToken = (): string | null => {
  return localStorage.getItem('fluxbase_impersonation_token')
}

// Helper to get active token (impersonation takes precedence)
const getActiveToken = (): string | null => {
  return getImpersonationToken() || getAccessToken()
}

// Create axios instance with default config
const api: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 30000, // 30 seconds
})

// Request interceptor to add auth token (with impersonation support)
api.interceptors.request.use(
  (config) => {
    // Use impersonation token if active, otherwise use admin token
    const accessToken = getActiveToken()
    if (accessToken) {
      config.headers.Authorization = `Bearer ${accessToken}`
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

api.interceptors.response.use(
  (response) => response,
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
        return Promise.reject(error)
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

        return Promise.reject(refreshError)
      }
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
  metadata: Record<string, unknown> | null
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
  trigger_type: string
  status: string
  status_code?: number
  duration_ms?: number
  result?: string
  logs?: string
  error_message?: string
  executed_at: string
}

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

// RPC Functions API Types
export interface FunctionParam {
  name: string
  type: string
  mode: string
  has_default: boolean
  position: number
}

export interface RPCFunction {
  schema: string
  name: string
  description: string
  parameters: FunctionParam[]
  return_type: string
  is_set_of: boolean
  volatility: string
  language: string
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
    options: EdgeFunctionInvokeOptions = {}
  ): Promise<string> => {
    const { method = 'POST', headers = {}, body = '' } = options

    const response = await api.request({
      url: `/api/v1/functions/${name}/invoke`,
      method,
      data: body,
      headers: {
        'Content-Type': 'application/json',
        ...headers,
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

export interface ExecutionLog {
  id: number
  job_id: string
  line_number: number
  message: string
  created_at: string
}

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
  submitJob: async (data: SubmitJobRequest): Promise<Job> => {
    const response = await api.post<Job>('/api/v1/jobs/submit', data)
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
    const response = await api.get<{ jobs: Job[]; limit: number; offset: number }>(
      `/api/v1/admin/jobs/queue${queryString ? `?${queryString}` : ''}`
    )
    return response.data.jobs
  },

  // Get job details
  getJob: async (jobId: string): Promise<Job> => {
    const response = await api.get<Job>(`/api/v1/admin/jobs/queue/${jobId}`)
    return response.data
  },

  // Get job execution logs
  getJobLogs: async (jobId: string, afterLine?: number): Promise<ExecutionLog[]> => {
    const params = afterLine !== undefined ? `?after=${afterLine}` : ''
    const response = await api.get<{ logs: ExecutionLog[] }>(
      `/api/v1/admin/jobs/queue/${jobId}/logs${params}`
    )
    return response.data.logs || []
  },

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
    const response = await api.get<JobStats>(`/api/v1/admin/jobs/stats${params}`)
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

// RPC Functions API
export const rpcApi = {
  // List all RPC functions
  list: async (): Promise<RPCFunction[]> => {
    const response = await api.get<RPCFunction[]>('/api/v1/rpc/')
    return response.data
  },

  // Execute RPC function
  execute: async (
    schema: string,
    name: string,
    params: Record<string, unknown>
  ): Promise<unknown> => {
    const path =
      schema === 'public'
        ? `/api/v1/rpc/${name}`
        : `/api/v1/rpc/${schema}/${name}`
    const response = await api.post(path, params)
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

// API Keys API Types
export interface APIKey {
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

export interface CreateAPIKeyRequest {
  name: string
  description?: string
  scopes: string[]
  rate_limit_per_minute: number
  expires_at?: string
}

export interface CreateAPIKeyResponse {
  api_key: APIKey
  key: string
}

// API Keys API
export const apiKeysApi = {
  // List all API keys
  list: async (): Promise<APIKey[]> => {
    const response = await api.get<APIKey[]>('/api/v1/api-keys')
    return response.data
  },

  // Create API key
  create: async (
    request: CreateAPIKeyRequest
  ): Promise<CreateAPIKeyResponse> => {
    const response = await api.post<CreateAPIKeyResponse>(
      '/api/v1/api-keys',
      request
    )
    return response.data
  },

  // Revoke API key
  revoke: async (id: string): Promise<{ message: string }> => {
    const response = await api.post<{ message: string }>(
      `/api/v1/api-keys/${id}/revoke`
    )
    return response.data
  },

  // Delete API key
  delete: async (id: string): Promise<{ message: string }> => {
    const response = await api.delete<{ message: string }>(
      `/api/v1/api-keys/${id}`
    )
    return response.data
  },
}

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
}

// OAuth Provider Management Types
export interface OAuthProviderConfig {
  id: string
  provider_name: string
  display_name: string
  enabled: boolean
  client_id: string
  client_secret?: string
  redirect_url: string
  scopes: string[]
  is_custom: boolean
  authorization_url?: string
  token_url?: string
  user_info_url?: string
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
