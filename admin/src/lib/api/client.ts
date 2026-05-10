import axios, {
  type AxiosError,
  type AxiosInstance,
  type AxiosRequestConfig,
  type AxiosResponse,
} from "axios";
import { useAuthStore } from "@/stores/auth-store";
import { useTenantStore } from "@/stores/tenant-store";
import { useBranchStore } from "@/stores/branch-store";

export const API_BASE_URL =
  window.__FLUXBASE_CONFIG__?.publicBaseURL ||
  import.meta.env.VITE_API_URL ||
  "";

/**
 * Axios-based API client for admin dashboard API calls.
 *
 * Use this for all /api/v1/admin/* and /dashboard/auth/* endpoints.
 * Automatically includes Authorization, X-FB-Tenant, and X-Fluxbase-Branch headers.
 *
 * For realtime subscriptions and user-facing auth (signUp/signOut), use the
 * Fluxbase SDK client from '@/lib/fluxbase-client' instead.
 */
export const api: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    "Content-Type": "application/json",
  },
  timeout: 30000,
});

let isRefreshing = false;
let failedQueue: Array<{
  resolve: (value: unknown) => void;
  reject: (reason?: unknown) => void;
}> = [];

const processQueue = (error: Error | null, token: string | null = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else {
      prom.resolve(token);
    }
  });
  failedQueue = [];
};

const isNotLoggedInResponse = (data: unknown): boolean => {
  if (!data || typeof data !== "object") return false;
  const obj = data as Record<string, unknown>;
  const errorFields = [obj.error, obj.message, obj.msg, obj.detail];
  for (const field of errorFields) {
    if (typeof field === "string") {
      const lower = field.toLowerCase();
      if (
        lower.includes("not logged in") ||
        lower.includes("not authenticated") ||
        lower.includes("unauthorized") ||
        lower.includes("invalid token") ||
        lower.includes("token expired") ||
        lower.includes("session expired") ||
        lower.includes("authentication required")
      ) {
        return true;
      }
    }
  }
  return false;
};

const SKIP_REFRESH_PATHS = ["/api/v1/admin/branches"];

async function refreshTokens(): Promise<string> {
  const { refreshToken } = useAuthStore.getState().auth;
  if (!refreshToken) {
    throw new Error("No refresh token available");
  }

  const response = await axios.post(`${API_BASE_URL}/api/v1/admin/refresh`, {
    refresh_token: refreshToken,
  });

  const {
    access_token,
    refresh_token: newRefreshToken,
    user,
    expires_in,
  } = response.data;

  const store = useAuthStore.getState().auth;
  store.setTokens(access_token, newRefreshToken);
  if (user) {
    store.setUser({
      accountNo: user.id,
      email: user.email,
      role: [user.role || "tenant_admin"],
      exp: Date.now() + expires_in * 1000,
    });
  }

  return access_token;
}

function forceLogout(): Promise<never> {
  useAuthStore.getState().auth.reset();
  window.location.href = "/admin/login";
  return new Promise(() => {});
}

async function refreshAndRetry(
  originalConfig: AxiosRequestConfig & { _retry?: boolean },
): Promise<AxiosResponse> {
  if (originalConfig._retry) {
    return forceLogout();
  }

  originalConfig._retry = true;

  if (isRefreshing) {
    return new Promise((resolve, reject) => {
      failedQueue.push({ resolve, reject });
    }).then((token) => {
      if (originalConfig.headers) {
        originalConfig.headers.Authorization = `Bearer ${token}`;
      }
      return api(originalConfig);
    });
  }

  isRefreshing = true;

  try {
    const accessToken = await refreshTokens();
    processQueue(null, accessToken);
    isRefreshing = false;
    if (originalConfig.headers) {
      originalConfig.headers.Authorization = `Bearer ${accessToken}`;
    }
    return api(originalConfig);
  } catch (refreshError) {
    processQueue(refreshError as Error, null);
    isRefreshing = false;
    return forceLogout();
  }
}

api.interceptors.request.use(
  (config) => {
    if (!config.headers.Authorization) {
      const { accessToken } = useAuthStore.getState().auth;
      if (accessToken) {
        config.headers.Authorization = `Bearer ${accessToken}`;
      }
    }

    try {
      const currentTenant = useTenantStore.getState().currentTenant;
      if (currentTenant?.id) {
        config.headers["X-FB-Tenant"] = currentTenant.id;
      }
    } catch {
      /* tenant store may not be available */
    }

    try {
      const currentBranch = useBranchStore.getState().currentBranch;
      if (currentBranch?.slug && currentBranch.type !== "main") {
        config.headers["X-Fluxbase-Branch"] = currentBranch.slug;
      }
    } catch {
      /* branch store may not be available */
    }

    return config;
  },
  (error) => Promise.reject(error),
);

api.interceptors.response.use(
  (response) => {
    if (isNotLoggedInResponse(response.data)) {
      return refreshAndRetry(response.config);
    }
    return response;
  },
  async (error: AxiosError) => {
    const originalConfig = error.config;
    const url = originalConfig?.url || "";
    const shouldSkipRefresh = SKIP_REFRESH_PATHS.some((path) =>
      url.startsWith(path),
    );

    if (shouldSkipRefresh) {
      return Promise.reject(error);
    }

    if (error.response?.status === 401 && originalConfig) {
      return refreshAndRetry(originalConfig);
    }

    if (
      error.response?.data &&
      isNotLoggedInResponse(error.response.data) &&
      originalConfig
    ) {
      return refreshAndRetry(originalConfig);
    }

    return Promise.reject(error);
  },
);

export const getDashboardAuthHeaders = (): Record<string, string> => {
  const { accessToken } = useAuthStore.getState().auth;
  return accessToken ? { Authorization: `Bearer ${accessToken}` } : {};
};
