/**
 * Database Access Edge Function Example
 *
 * This function demonstrates how to access the Fluxbase database
 * from within an edge function using the Fluxbase SDK.
 *
 * Now using the proper SDK instead of direct fetch calls!
 *
 * Note: Requires authentication (user must be logged in)
 */

import { createClient } from '@fluxbase/sdk'
import { corsHeaders } from '_shared/cors.ts'

interface Request {
  method: string
  url: string
  headers: Record<string, string>
  body: string
  user_id?: string
  user_email?: string
  user_role?: string
  session_id?: string
}

async function handler(req: Request) {
  // Handle CORS preflight
  if (req.method === 'OPTIONS') {
    return {
      status: 204,
      headers: corsHeaders(),
      body: '',
    }
  }

  try {
    // Check authentication
    if (!req.user_id) {
      return {
        status: 401,
        headers: { ...corsHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify({
          error: 'Unauthorized',
          message: 'This function requires authentication',
        }),
      }
    }

    // Create Fluxbase client with service role access
    const fluxbaseUrl = Deno.env.get('FLUXBASE_BASE_URL')
    const serviceKey = Deno.env.get('FLUXBASE_SERVICE_ROLE_KEY')

    if (!fluxbaseUrl || !serviceKey) {
      return {
        status: 500,
        headers: { ...corsHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify({
          error: 'Configuration Error',
          message: 'Fluxbase configuration not available',
        }),
      }
    }

    // Create SDK client
    const client = createClient(fluxbaseUrl, serviceKey)

    // Parse request body
    const data = req.body ? JSON.parse(req.body) : {}
    const table = data.table || 'users'
    const limit = data.limit || 10

    // Query database using SDK
    // This is much cleaner than direct fetch calls!
    const { data: results, error } = await client
      .from(table)
      .select('*')
      .limit(limit)
      .execute()

    if (error) {
      console.error('Database query error:', error)
      return {
        status: 500,
        headers: { ...corsHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify({
          error: 'Database Error',
          message: error.message || 'Failed to query database',
        }),
      }
    }

    return {
      status: 200,
      headers: { ...corsHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify({
        success: true,
        user: {
          id: req.user_id,
          email: req.user_email,
          role: req.user_role,
          session_id: req.session_id,
        },
        query: {
          table,
          limit,
          count: results?.length || 0,
        },
        results: results,
        timestamp: new Date().toISOString(),
      }),
    }
  } catch (error) {
    console.error('Database access error:', error)

    return {
      status: 500,
      headers: { ...corsHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify({
        error: 'Internal Server Error',
        message: error instanceof Error ? error.message : 'An unexpected error occurred',
      }),
    }
  }
}
