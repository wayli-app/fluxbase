/**
 * Fluxbase SDK client configuration for Admin UI
 */

import { createClient } from '@fluxbase/sdk'
import { getAccessToken } from './auth'

// Declare the runtime config type injected by the server
declare global {
  interface Window {
    __FLUXBASE_CONFIG__?: {
      publicBaseURL?: string
    }
  }
}

// Base URL for the API - priority order:
// 1. Runtime config injected by server (FLUXBASE_PUBLIC_BASE_URL)
// 2. Build-time environment variable (VITE_API_URL)
// 3. Current origin (works when dashboard is served from the same domain)
const API_BASE_URL =
  window.__FLUXBASE_CONFIG__?.publicBaseURL ||
  import.meta.env.VITE_API_URL ||
  window.location.origin
const API_KEY = import.meta.env.VITE_API_KEY || 'anonymous'

// Helper to get impersonation token if active
const getImpersonationToken = (): string | null => {
  return localStorage.getItem('fluxbase_impersonation_token')
}

// Helper to get active token (impersonation takes precedence over admin token)
const getActiveToken = (): string | null => {
  return getImpersonationToken() || getAccessToken()
}

// Create the Fluxbase client
export const fluxbaseClient = createClient(API_BASE_URL, API_KEY, {
  auth: {
    autoRefresh: false, // Disable auto-refresh since we manage tokens ourselves
    persist: false, // We manage persistence ourselves via localStorage
  },
  timeout: 30000, // 30 seconds
})

// Initialize SDK with existing tokens on load (checks for impersonation token first)
const existingToken = getActiveToken()
if (existingToken) {
  fluxbaseClient.setAuthToken(existingToken)
}

// Helper to set the auth token
export function setAuthToken(token: string | null) {
  if (token) {
    fluxbaseClient.setAuthToken(token)
  } else {
    fluxbaseClient.setAuthToken(null)
  }
}

// Helper to get the auth token
export function getAuthToken(): string | null {
  return fluxbaseClient.getAuthToken()
}

// Helper to sync SDK token with current active token (impersonation or admin)
export function syncAuthToken() {
  const activeToken = getActiveToken()
  setAuthToken(activeToken)
}

export default fluxbaseClient
