/**
 * Error Handling Edge Function Example
 *
 * This function demonstrates proper error handling patterns:
 * - Input validation
 * - Try-catch blocks
 * - Structured error responses
 * - Different error types
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
  try {
    // Parse request body
    let data;
    try {
      data = req.body ? JSON.parse(req.body) : {};
    } catch (parseError) {
      return {
        status: 400,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          error: 'Invalid JSON',
          message: 'Request body must be valid JSON',
          details: parseError instanceof Error ? parseError.message : 'Unknown error',
        }),
      };
    }

    // Validate required fields
    if (!data.action) {
      return {
        status: 400,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          error: 'Validation Error',
          message: 'Missing required field: action',
          required_fields: ['action'],
        }),
      };
    }

    // Simulate different actions with potential errors
    switch (data.action) {
      case 'success':
        return {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            success: true,
            message: 'Operation completed successfully',
          }),
        };

      case 'simulate_error':
        throw new Error('This is a simulated error for testing');

      case 'divide':
        const { numerator, denominator } = data;

        if (numerator === undefined || denominator === undefined) {
          return {
            status: 400,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              error: 'Missing Parameters',
              message: 'Both numerator and denominator are required',
            }),
          };
        }

        if (denominator === 0) {
          return {
            status: 400,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              error: 'Division by Zero',
              message: 'Cannot divide by zero',
            }),
          };
        }

        return {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            result: numerator / denominator,
            operation: 'division',
          }),
        };

      case 'timeout':
        // Simulate a long-running operation
        await new Promise(resolve => setTimeout(resolve, 35000)); // 35 seconds (exceeds default 30s timeout)
        return {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ message: 'This should timeout' }),
        };

      default:
        return {
          status: 400,
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            error: 'Invalid Action',
            message: `Unknown action: ${data.action}`,
            valid_actions: ['success', 'simulate_error', 'divide', 'timeout'],
          }),
        };
    }
  } catch (error) {
    // Global error handler
    console.error('Edge function error:', error);

    return {
      status: 500,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        error: 'Internal Server Error',
        message: error instanceof Error ? error.message : 'An unexpected error occurred',
        stack: error instanceof Error ? error.stack : undefined,
      }),
    };
  }
}
