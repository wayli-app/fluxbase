/**
 * Multi-file edge function example
 *
 * This function demonstrates:
 * - Directory-based function with multiple files
 * - Local imports from supporting files (./types.ts, ./helpers.ts)
 * - Shared module imports from _shared/cors.ts
 * - Type-safe webhook processing
 */

import { corsHeaders, handleCors } from '_shared/cors.ts';
import { WebhookPayload } from './types.ts';
import { validateWebhook, processWebhook } from './helpers.ts';

async function handler(req: any) {
  // Handle CORS preflight
  const corsResponse = handleCors(req);
  if (corsResponse) return corsResponse;

  // Parse webhook payload
  let payload: WebhookPayload;
  try {
    payload = JSON.parse(req.body || '{}');
  } catch (error) {
    return {
      status: 400,
      headers: corsHeaders(),
      body: JSON.stringify({ error: 'Invalid JSON payload' }),
    };
  }

  // Validate webhook structure
  if (!validateWebhook(payload)) {
    return {
      status: 400,
      headers: corsHeaders(),
      body: JSON.stringify({
        error: 'Invalid webhook payload',
        required: ['event', 'data', 'timestamp'],
      }),
    };
  }

  // Process the webhook
  const result = processWebhook(payload);

  return {
    status: 200,
    headers: corsHeaders(),
    body: JSON.stringify(result),
  };
}
