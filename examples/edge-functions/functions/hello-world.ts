/**
 * Simple Hello World Edge Function
 *
 * This function demonstrates the basic structure of an edge function.
 * It accepts a request and returns a JSON response.
 *
 * @fluxbase:allow-unauthenticated
 */

interface Request {
  method: string;
  url: string;
  headers: Record<string, string>;
  body: string;
  user_id?: string;
}

async function handler(req: Request) {
  // Parse the request body
  const data = req.body ? JSON.parse(req.body) : {};
  const name = data.name || 'World';

  // Return a JSON response
  return {
    status: 200,
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      message: `Hello, ${name}!`,
      timestamp: new Date().toISOString(),
      method: req.method,
      authenticated: !!req.user_id,
    }),
  };
}
