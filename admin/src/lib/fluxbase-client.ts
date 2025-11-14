/**
 * Fluxbase SDK client configuration for Admin UI
 */

import { createClient } from '@fluxbase/sdk'
import { getAccessToken } from './auth'

// Base URL for the API - can be overridden with environment variable
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'
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
    autoRefresh: true,
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

export default fluxbaseClient
