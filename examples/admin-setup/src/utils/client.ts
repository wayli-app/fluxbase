import { createClient, type FluxbaseClient } from '@fluxbase/sdk'
import dotenv from 'dotenv'

// Load environment variables
dotenv.config()

/**
 * Create and configure the Fluxbase client
 */
export function getClient(): FluxbaseClient {
  const url = process.env.FLUXBASE_URL || 'http://localhost:8080'

  const client = createClient({ url })

  return client
}

/**
 * Authenticate as admin
 */
export async function authenticateAdmin(client: FluxbaseClient): Promise<void> {
  const email = process.env.ADMIN_EMAIL
  const password = process.env.ADMIN_PASSWORD

  if (!email || !password) {
    throw new Error('ADMIN_EMAIL and ADMIN_PASSWORD must be set in .env file')
  }

  await client.admin.login({ email, password })
}

/**
 * Get environment variable with validation
 */
export function getEnv(key: string, required: boolean = false): string | undefined {
  const value = process.env[key]

  if (required && !value) {
    throw new Error(`Required environment variable ${key} is not set`)
  }

  return value
}
