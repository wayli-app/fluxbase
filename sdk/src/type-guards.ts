/**
 * Type guard utilities for Fluxbase SDK
 * Provides runtime type checking and type narrowing for response types
 */

import type {
  FluxbaseResponse,
  FluxbaseAuthResponse,
  PostgrestResponse,
  PostgrestError,
  AuthResponseData,
} from './types'

// ============================================================================
// FluxbaseResponse Type Guards
// ============================================================================

/**
 * Type guard to check if a FluxbaseResponse is an error response
 *
 * @param response - The response to check
 * @returns true if the response is an error (data is null, error is not null)
 *
 * @example
 * ```typescript
 * const result = await client.auth.signIn(credentials)
 *
 * if (isFluxbaseError(result)) {
 *   // TypeScript knows: result.error is Error, result.data is null
 *   console.error('Sign in failed:', result.error.message)
 *   return
 * }
 *
 * // TypeScript knows: result.data is T, result.error is null
 * console.log('Signed in as:', result.data.user.email)
 * ```
 */
export function isFluxbaseError<T>(
  response: FluxbaseResponse<T>
): response is { data: null; error: Error } {
  return response.error !== null
}

/**
 * Type guard to check if a FluxbaseResponse is a success response
 *
 * @param response - The response to check
 * @returns true if the response is successful (data is not null, error is null)
 *
 * @example
 * ```typescript
 * const result = await client.from('users').select('*').execute()
 *
 * if (isFluxbaseSuccess(result)) {
 *   // TypeScript knows: result.data is T, result.error is null
 *   result.data.forEach(user => console.log(user.name))
 * }
 * ```
 */
export function isFluxbaseSuccess<T>(
  response: FluxbaseResponse<T>
): response is { data: T; error: null } {
  return response.error === null
}

// ============================================================================
// FluxbaseAuthResponse Type Guards (specialized for auth)
// ============================================================================

/**
 * Type guard to check if an auth response is an error
 *
 * @param response - The auth response to check
 * @returns true if the auth operation failed
 *
 * @example
 * ```typescript
 * const result = await client.auth.signUp(credentials)
 *
 * if (isAuthError(result)) {
 *   console.error('Sign up failed:', result.error.message)
 *   return
 * }
 *
 * // TypeScript knows result.data contains user and session
 * console.log('Welcome,', result.data.user.email)
 * ```
 */
export function isAuthError(
  response: FluxbaseAuthResponse
): response is { data: null; error: Error } {
  return response.error !== null
}

/**
 * Type guard to check if an auth response is successful
 *
 * @param response - The auth response to check
 * @returns true if the auth operation succeeded
 */
export function isAuthSuccess(
  response: FluxbaseAuthResponse
): response is { data: AuthResponseData; error: null } {
  return response.error === null
}

// ============================================================================
// PostgrestResponse Type Guards
// ============================================================================

/**
 * Type guard to check if a PostgrestResponse has an error
 *
 * @param response - The Postgrest response to check
 * @returns true if the response contains an error
 *
 * @example
 * ```typescript
 * const response = await client.from('products').select('*').execute()
 *
 * if (hasPostgrestError(response)) {
 *   // TypeScript knows: response.error is PostgrestError
 *   console.error('Query failed:', response.error.message)
 *   if (response.error.hint) {
 *     console.log('Hint:', response.error.hint)
 *   }
 *   return
 * }
 *
 * // TypeScript knows: response.data is T (not null)
 * console.log('Found', response.data.length, 'products')
 * ```
 */
export function hasPostgrestError<T>(
  response: PostgrestResponse<T>
): response is PostgrestResponse<T> & { error: PostgrestError; data: null } {
  return response.error !== null
}

/**
 * Type guard to check if a PostgrestResponse is successful (has data)
 *
 * @param response - The Postgrest response to check
 * @returns true if the response has data and no error
 */
export function isPostgrestSuccess<T>(
  response: PostgrestResponse<T>
): response is PostgrestResponse<T> & { data: T; error: null } {
  return response.error === null && response.data !== null
}

// ============================================================================
// Utility Type Guards for unknown narrowing
// ============================================================================

/**
 * Type guard to check if a value is a non-null object
 * Useful for narrowing unknown types from API responses
 *
 * @param value - The value to check
 * @returns true if value is a non-null object
 */
export function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

/**
 * Type guard to check if a value is an array
 * Useful for narrowing unknown types from API responses
 *
 * @param value - The value to check
 * @returns true if value is an array
 */
export function isArray(value: unknown): value is unknown[] {
  return Array.isArray(value)
}

/**
 * Type guard to check if a value is a string
 *
 * @param value - The value to check
 * @returns true if value is a string
 */
export function isString(value: unknown): value is string {
  return typeof value === 'string'
}

/**
 * Type guard to check if a value is a number
 *
 * @param value - The value to check
 * @returns true if value is a number (excludes NaN)
 */
export function isNumber(value: unknown): value is number {
  return typeof value === 'number' && !Number.isNaN(value)
}

/**
 * Type guard to check if a value is a boolean
 *
 * @param value - The value to check
 * @returns true if value is a boolean
 */
export function isBoolean(value: unknown): value is boolean {
  return typeof value === 'boolean'
}

/**
 * Assert that a value is of type T, throwing if validation fails
 *
 * @param value - The value to assert
 * @param validator - A type guard function to validate the value
 * @param errorMessage - Optional custom error message
 * @throws Error if validation fails
 *
 * @example
 * ```typescript
 * const response = await client.functions.invoke('get-user')
 * assertType(response.data, isObject, 'Expected user object')
 * // Now response.data is typed as Record<string, unknown>
 * ```
 */
export function assertType<T>(
  value: unknown,
  validator: (v: unknown) => v is T,
  errorMessage = 'Type assertion failed'
): asserts value is T {
  if (!validator(value)) {
    throw new Error(errorMessage)
  }
}
