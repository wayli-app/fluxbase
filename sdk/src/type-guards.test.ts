import { describe, it, expect } from 'vitest'
import {
  isFluxbaseError,
  isFluxbaseSuccess,
  isAuthError,
  isAuthSuccess,
  hasPostgrestError,
  isPostgrestSuccess,
  isObject,
  isArray,
  isString,
  isNumber,
  isBoolean,
  assertType,
} from './type-guards'
import type {
  FluxbaseResponse,
  FluxbaseAuthResponse,
  PostgrestResponse,
  User,
} from './types'

describe('FluxbaseResponse Type Guards', () => {
  describe('isFluxbaseError', () => {
    it('returns true for error responses', () => {
      const response: FluxbaseResponse<string> = {
        data: null,
        error: new Error('Something went wrong'),
      }
      expect(isFluxbaseError(response)).toBe(true)
    })

    it('returns false for success responses', () => {
      const response: FluxbaseResponse<string> = {
        data: 'success',
        error: null,
      }
      expect(isFluxbaseError(response)).toBe(false)
    })

    it('narrows type correctly on error', () => {
      const response: FluxbaseResponse<{ id: number }> = {
        data: null,
        error: new Error('Failed'),
      }

      if (isFluxbaseError(response)) {
        // TypeScript should know error is Error and data is null
        expect(response.error.message).toBe('Failed')
        expect(response.data).toBeNull()
      }
    })
  })

  describe('isFluxbaseSuccess', () => {
    it('returns true for success responses', () => {
      const response: FluxbaseResponse<{ id: number }> = {
        data: { id: 1 },
        error: null,
      }
      expect(isFluxbaseSuccess(response)).toBe(true)
    })

    it('returns false for error responses', () => {
      const response: FluxbaseResponse<{ id: number }> = {
        data: null,
        error: new Error('Failed'),
      }
      expect(isFluxbaseSuccess(response)).toBe(false)
    })

    it('narrows type correctly on success', () => {
      const response: FluxbaseResponse<{ id: number; name: string }> = {
        data: { id: 1, name: 'Test' },
        error: null,
      }

      if (isFluxbaseSuccess(response)) {
        // TypeScript should know data is the correct type
        expect(response.data.id).toBe(1)
        expect(response.data.name).toBe('Test')
        expect(response.error).toBeNull()
      }
    })
  })
})

describe('FluxbaseAuthResponse Type Guards', () => {
  const mockUser: User = {
    id: 'user-123',
    email: 'test@example.com',
    email_verified: true,
    role: 'user',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  }

  describe('isAuthError', () => {
    it('returns true for auth error responses', () => {
      const response: FluxbaseAuthResponse = {
        data: null,
        error: new Error('Invalid credentials'),
      }
      expect(isAuthError(response)).toBe(true)
    })

    it('returns false for auth success responses', () => {
      const response: FluxbaseAuthResponse = {
        data: {
          user: mockUser,
          session: {
            user: mockUser,
            access_token: 'token',
            refresh_token: 'refresh',
            expires_in: 3600,
          },
        },
        error: null,
      }
      expect(isAuthError(response)).toBe(false)
    })
  })

  describe('isAuthSuccess', () => {
    it('returns true for auth success responses', () => {
      const response: FluxbaseAuthResponse = {
        data: {
          user: mockUser,
          session: {
            user: mockUser,
            access_token: 'token',
            refresh_token: 'refresh',
            expires_in: 3600,
          },
        },
        error: null,
      }
      expect(isAuthSuccess(response)).toBe(true)
    })

    it('returns false for auth error responses', () => {
      const response: FluxbaseAuthResponse = {
        data: null,
        error: new Error('Auth failed'),
      }
      expect(isAuthSuccess(response)).toBe(false)
    })
  })
})

describe('PostgrestResponse Type Guards', () => {
  describe('hasPostgrestError', () => {
    it('returns true when error exists', () => {
      const response: PostgrestResponse<unknown[]> = {
        data: null,
        error: { message: 'Not found', code: '404' },
        count: null,
        status: 404,
        statusText: 'Not Found',
      }
      expect(hasPostgrestError(response)).toBe(true)
    })

    it('returns false when no error', () => {
      const response: PostgrestResponse<unknown[]> = {
        data: [],
        error: null,
        count: 0,
        status: 200,
        statusText: 'OK',
      }
      expect(hasPostgrestError(response)).toBe(false)
    })

    it('narrows type correctly on error', () => {
      const response: PostgrestResponse<{ id: number }[]> = {
        data: null,
        error: { message: 'Query failed', hint: 'Check your SQL' },
        count: null,
        status: 400,
        statusText: 'Bad Request',
      }

      if (hasPostgrestError(response)) {
        expect(response.error.message).toBe('Query failed')
        expect(response.error.hint).toBe('Check your SQL')
      }
    })
  })

  describe('isPostgrestSuccess', () => {
    it('returns true for successful queries', () => {
      const response: PostgrestResponse<{ id: number }[]> = {
        data: [{ id: 1 }, { id: 2 }],
        error: null,
        count: 2,
        status: 200,
        statusText: 'OK',
      }
      expect(isPostgrestSuccess(response)).toBe(true)
    })

    it('returns false when error exists', () => {
      const response: PostgrestResponse<unknown[]> = {
        data: null,
        error: { message: 'Error' },
        count: null,
        status: 500,
        statusText: 'Internal Server Error',
      }
      expect(isPostgrestSuccess(response)).toBe(false)
    })

    it('narrows type correctly on success', () => {
      const response: PostgrestResponse<{ id: number; name: string }[]> = {
        data: [{ id: 1, name: 'Alice' }],
        error: null,
        count: 1,
        status: 200,
        statusText: 'OK',
      }

      if (isPostgrestSuccess(response)) {
        expect(response.data[0].name).toBe('Alice')
        expect(response.error).toBeNull()
      }
    })
  })
})

describe('Utility Type Guards', () => {
  describe('isObject', () => {
    it('returns true for plain objects', () => {
      expect(isObject({})).toBe(true)
      expect(isObject({ a: 1 })).toBe(true)
      expect(isObject({ nested: { value: true } })).toBe(true)
    })

    it('returns false for arrays', () => {
      expect(isObject([])).toBe(false)
      expect(isObject([1, 2, 3])).toBe(false)
    })

    it('returns false for null', () => {
      expect(isObject(null)).toBe(false)
    })

    it('returns false for primitives', () => {
      expect(isObject('string')).toBe(false)
      expect(isObject(123)).toBe(false)
      expect(isObject(true)).toBe(false)
      expect(isObject(undefined)).toBe(false)
    })
  })

  describe('isArray', () => {
    it('returns true for arrays', () => {
      expect(isArray([])).toBe(true)
      expect(isArray([1, 2, 3])).toBe(true)
      expect(isArray(['a', 'b'])).toBe(true)
    })

    it('returns false for objects', () => {
      expect(isArray({})).toBe(false)
      expect(isArray({ length: 0 })).toBe(false)
    })

    it('returns false for primitives', () => {
      expect(isArray('string')).toBe(false)
      expect(isArray(123)).toBe(false)
    })
  })

  describe('isString', () => {
    it('returns true for strings', () => {
      expect(isString('')).toBe(true)
      expect(isString('hello')).toBe(true)
      expect(isString(`template`)).toBe(true)
    })

    it('returns false for non-strings', () => {
      expect(isString(123)).toBe(false)
      expect(isString(null)).toBe(false)
      expect(isString(undefined)).toBe(false)
      expect(isString({})).toBe(false)
    })
  })

  describe('isNumber', () => {
    it('returns true for valid numbers', () => {
      expect(isNumber(0)).toBe(true)
      expect(isNumber(42)).toBe(true)
      expect(isNumber(-3.14)).toBe(true)
      expect(isNumber(Infinity)).toBe(true)
    })

    it('returns false for NaN', () => {
      expect(isNumber(NaN)).toBe(false)
    })

    it('returns false for non-numbers', () => {
      expect(isNumber('123')).toBe(false)
      expect(isNumber(null)).toBe(false)
      expect(isNumber(undefined)).toBe(false)
    })
  })

  describe('isBoolean', () => {
    it('returns true for booleans', () => {
      expect(isBoolean(true)).toBe(true)
      expect(isBoolean(false)).toBe(true)
    })

    it('returns false for truthy/falsy non-booleans', () => {
      expect(isBoolean(0)).toBe(false)
      expect(isBoolean(1)).toBe(false)
      expect(isBoolean('')).toBe(false)
      expect(isBoolean('true')).toBe(false)
      expect(isBoolean(null)).toBe(false)
    })
  })

  describe('assertType', () => {
    it('does not throw for valid types', () => {
      expect(() => assertType('hello', isString)).not.toThrow()
      expect(() => assertType(123, isNumber)).not.toThrow()
      expect(() => assertType({}, isObject)).not.toThrow()
    })

    it('throws for invalid types with default message', () => {
      expect(() => assertType(123, isString)).toThrow('Type assertion failed')
    })

    it('throws for invalid types with custom message', () => {
      expect(() => assertType(123, isString, 'Expected a string value')).toThrow(
        'Expected a string value'
      )
    })

    it('narrows type after assertion', () => {
      const value: unknown = { id: 1, name: 'Test' }
      assertType(value, isObject)
      // After assertion, value is Record<string, unknown>
      expect(value.id).toBe(1)
      expect(value.name).toBe('Test')
    })
  })
})
