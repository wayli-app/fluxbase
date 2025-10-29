/**
 * Fluxbase SDK client configuration for Admin UI
 */

import { createClient } from '@fluxbase/sdk'
import { getAccessToken } from './auth'

// Base URL for the API - can be overridden with environment variable
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

// Create the Fluxbase client
export const fluxbaseClient = createClient({
  url: API_BASE_URL,
  auth: {
    autoRefresh: true,
    persist: false, // We manage persistence ourselves via localStorage
  },
  timeout: 30000, // 30 seconds
})

// Initialize SDK with existing tokens on load
const existingToken = getAccessToken()
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
