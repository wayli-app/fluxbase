import type { FluxbaseResponse, VoidResponse } from '../types'

/**
 * Wraps an async operation with try-catch, returning Fluxbase response format
 * @param operation The async operation to wrap
 * @returns Promise resolving to { data, error: null } on success or { data: null, error } on failure
 */
export async function wrapAsync<T>(
  operation: () => Promise<T>
): Promise<FluxbaseResponse<T>> {
  try {
    const data = await operation()
    return { data, error: null }
  } catch (error) {
    return {
      data: null,
      error: error instanceof Error ? error : new Error(String(error))
    }
  }
}

/**
 * Wraps void async operations (operations that don't return data)
 * @param operation The async void operation to wrap
 * @returns Promise resolving to { error: null } on success or { error } on failure
 */
export async function wrapAsyncVoid(
  operation: () => Promise<void>
): Promise<VoidResponse> {
  try {
    await operation()
    return { error: null }
  } catch (error) {
    return {
      error: error instanceof Error ? error : new Error(String(error))
    }
  }
}

/**
 * Wraps synchronous operations (for consistency)
 * @param operation The sync operation to wrap
 * @returns { data, error: null } on success or { data: null, error } on failure
 */
export function wrapSync<T>(
  operation: () => T
): FluxbaseResponse<T> {
  try {
    const data = operation()
    return { data, error: null }
  } catch (error) {
    return {
      data: null,
      error: error instanceof Error ? error : new Error(String(error))
    }
  }
}
