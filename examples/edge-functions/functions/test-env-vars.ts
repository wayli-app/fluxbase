// Test Edge Function - Environment Variable Access
// This function tests access to server environment variables

interface Request {
  method: string;
  url: string;
  headers: Record<string, string>;
  body: string;
}

async function handler(req: Request) {
  // Test accessing environment variables
  const envVars = {
    // These should be available
    FLUXBASE_BASE_URL: Deno.env.get("FLUXBASE_BASE_URL"),
    FLUXBASE_SERVICE_ROLE_KEY: Deno.env.get("FLUXBASE_SERVICE_ROLE_KEY") ? "[PRESENT]" : "[MISSING]",
    FLUXBASE_ANON_KEY: Deno.env.get("FLUXBASE_ANON_KEY") ? "[PRESENT]" : "[MISSING]",

    // These should be blocked for security
    FLUXBASE_AUTH_JWT_SECRET: Deno.env.get("FLUXBASE_AUTH_JWT_SECRET") ? "[LEAKED!]" : "[BLOCKED]",
    FLUXBASE_DATABASE_PASSWORD: Deno.env.get("FLUXBASE_DATABASE_PASSWORD") ? "[LEAKED!]" : "[BLOCKED]",

    // Test that we can get all FLUXBASE_* vars
    all_fluxbase_vars: {} as Record<string, string>
  };

  // Try to enumerate all environment variables (only works if allow_env is true)
  try {
    const allEnv = Deno.env.toObject();
    for (const [key, value] of Object.entries(allEnv)) {
      if (key.startsWith("FLUXBASE_")) {
        // Don't leak actual secret values
        if (key.includes("SECRET") || key.includes("PASSWORD") || key.includes("KEY")) {
          envVars.all_fluxbase_vars[key] = value ? "[PRESENT]" : "[MISSING]";
        } else {
          envVars.all_fluxbase_vars[key] = value;
        }
      }
    }
  } catch (error) {
    envVars.all_fluxbase_vars = { error: "Failed to enumerate env vars" };
  }

  return {
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      message: "Environment variable access test",
      environment: envVars,
      timestamp: new Date().toISOString()
    }, null, 2)
  };
}

export { handler };
