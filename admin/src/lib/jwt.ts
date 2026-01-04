// JWT decoder utility for extracting user information from tokens
// Note: This is a simple decoder that only extracts the payload.
// It does NOT verify the signature - that's done by the backend.

/**
 * JWT payload interface
 */
export interface JWTPayload {
  user_id: string
  email: string
  name?: string
  role: string
  session_id: string
  token_type: string
  is_anonymous?: boolean
  user_metadata?: {
    name?: string
    avatar?: string
    [key: string]: unknown
  }
  app_metadata?: Record<string, unknown>
  iss?: string
  sub?: string
  iat?: number
  exp?: number
  nbf?: number
  jti?: string
}

/**
 * Decodes a JWT token and returns the payload
 * @param token - The JWT token string
 * @returns The decoded payload object
 * @throws Error if the token format is invalid
 */
export function decodeJWT(token: string): JWTPayload {
  const parts = token.split('.')
  if (parts.length !== 3) {
    throw new Error('Invalid JWT token format')
  }

  // Get the payload (second part)
  const payload = parts[1]

  // Decode base64url to base64
  const base64 = payload.replace(/-/g, '+').replace(/_/g, '/')

  // Decode base64 and parse JSON
  try {
    const decoded = atob(base64)
    return JSON.parse(decoded) as JWTPayload
  } catch (_error) {
    throw new Error('Failed to decode JWT payload')
  }
}
