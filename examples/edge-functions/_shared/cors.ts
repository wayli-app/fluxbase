/**
 * Shared CORS utilities for edge functions
 * This module can be imported by any edge function using:
 * import { corsHeaders, handleCors } from '_shared/cors.ts';
 */

export interface CorsConfig {
  origin?: string;
  methods?: string[];
  headers?: string[];
}

/**
 * Generate CORS headers for responses
 */
export function corsHeaders(config?: CorsConfig) {
  const origin = config?.origin || '*';
  const methods = config?.methods || ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'];
  const headers = config?.headers || [
    'authorization',
    'x-client-info',
    'apikey',
    'content-type',
  ];

  return {
    'Access-Control-Allow-Origin': origin,
    'Access-Control-Allow-Methods': methods.join(', '),
    'Access-Control-Allow-Headers': headers.join(', '),
  };
}

/**
 * Handle CORS preflight requests
 * Returns a response for OPTIONS requests, or null for other methods
 */
export function handleCors(req: any, config?: CorsConfig) {
  if (req.method === 'OPTIONS') {
    return {
      status: 200,
      headers: corsHeaders(config),
      body: '',
    };
  }
  return null;
}
