/**
 * Example edge function using shared modules
 *
 * This function demonstrates:
 * - Importing from _shared/cors.ts
 * - Importing from _shared/validation.ts
 * - Handling CORS preflight requests
 * - Input validation using shared utilities
 */

import { corsHeaders, handleCors } from '_shared/cors.ts';
import { validateEmail, validateRequired } from '_shared/validation.ts';

async function handler(req: any) {
  // Handle CORS preflight
  const corsResponse = handleCors(req);
  if (corsResponse) return corsResponse;

  // Parse request body
  let data;
  try {
    data = JSON.parse(req.body || '{}');
  } catch (error) {
    return {
      status: 400,
      headers: corsHeaders(),
      body: JSON.stringify({ error: 'Invalid JSON' }),
    };
  }

  // Validate required fields
  const missing = validateRequired(data, ['name', 'email']);
  if (missing.length > 0) {
    return {
      status: 400,
      headers: corsHeaders(),
      body: JSON.stringify({
        error: 'Missing required fields',
        fields: missing,
      }),
    };
  }

  // Validate email format
  if (!validateEmail(data.email)) {
    return {
      status: 400,
      headers: corsHeaders(),
      body: JSON.stringify({ error: 'Invalid email address' }),
    };
  }

  // Process the valid data
  const result = {
    message: `Hello ${data.name}!`,
    email: data.email,
    timestamp: new Date().toISOString(),
  };

  return {
    status: 200,
    headers: corsHeaders(),
    body: JSON.stringify(result),
  };
}
