/**
 * Authentication Tests
 */

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { FluxbaseAuth } from './auth'
import type { FluxbaseFetch } from './fetch'
import type { AuthResponse } from './types'

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {}
  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value
    },
    removeItem: (key: string) => {
      delete store[key]
    },
    clear: () => {
      store = {}
    },
  }
})()

Object.defineProperty(global, 'localStorage', { value: localStorageMock })

describe('FluxbaseAuth', () => {
  let mockFetch: FluxbaseFetch
  let auth: FluxbaseAuth

  beforeEach(() => {
    localStorageMock.clear()
    vi.clearAllTimers()

    mockFetch = {
      post: vi.fn(),
      get: vi.fn(),
      patch: vi.fn(),
      setAuthToken: vi.fn(),
    } as unknown as FluxbaseFetch

    auth = new FluxbaseAuth(mockFetch, true, true)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('initialization', () => {
    it('should initialize with no session', () => {
      expect(auth.getSession()).toBeNull()
      expect(auth.getUser()).toBeNull()
      expect(auth.getAccessToken()).toBeNull()
    })

    it('should restore session from localStorage', () => {
      const session = {
        access_token: 'test-token',
        refresh_token: 'refresh-token',
        expires_in: 3600,
        expires_at: Date.now() + 3600 * 1000,
        token_type: 'Bearer',
        user: { id: '1', email: 'test@example.com', created_at: '' },
      }

      localStorage.setItem('fluxbase.auth.session', JSON.stringify(session))

      const newAuth = new FluxbaseAuth(mockFetch, true, true)

      expect(newAuth.getSession()).toEqual(session)
      expect(mockFetch.setAuthToken).toHaveBeenCalledWith('test-token')
    })

    it('should ignore invalid stored session', () => {
      localStorage.setItem('fluxbase.auth.session', 'invalid-json')

      const newAuth = new FluxbaseAuth(mockFetch, true, true)

      expect(newAuth.getSession()).toBeNull()
      expect(localStorage.getItem('fluxbase.auth.session')).toBeNull()
    })
  })

  describe('signIn()', () => {
    it('should sign in successfully', async () => {
      const authResponse: AuthResponse = {
        access_token: 'new-access-token',
        refresh_token: 'new-refresh-token',
        expires_in: 3600,
        token_type: 'Bearer',
        user: { id: '1', email: 'user@example.com', created_at: new Date().toISOString() },
      }

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse)

      const session = await auth.signIn({
        email: 'user@example.com',
        password: 'password123',
      })

      expect(mockFetch.post).toHaveBeenCalledWith('/api/auth/signin', {
        email: 'user@example.com',
        password: 'password123',
      })
      expect(session.access_token).toBe('new-access-token')
      expect(session.user.email).toBe('user@example.com')
      expect(mockFetch.setAuthToken).toHaveBeenCalledWith('new-access-token')
    })

    it('should persist session to localStorage', async () => {
      const authResponse: AuthResponse = {
        access_token: 'token',
        refresh_token: 'refresh',
        expires_in: 3600,
        token_type: 'Bearer',
        user: { id: '1', email: 'user@example.com', created_at: '' },
      }

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse)

      await auth.signIn({ email: 'user@example.com', password: 'password' })

      const stored = localStorage.getItem('fluxbase.auth.session')
      expect(stored).toBeTruthy()
      expect(JSON.parse(stored!).access_token).toBe('token')
    })
  })

  describe('signUp()', () => {
    it('should sign up successfully', async () => {
      const authResponse: AuthResponse = {
        access_token: 'new-token',
        refresh_token: 'refresh-token',
        expires_in: 3600,
        token_type: 'Bearer',
        user: { id: '1', email: 'newuser@example.com', created_at: '' },
      }

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse)

      const session = await auth.signUp({
        email: 'newuser@example.com',
        password: 'password123',
      })

      expect(mockFetch.post).toHaveBeenCalledWith('/api/auth/signup', {
        email: 'newuser@example.com',
        password: 'password123',
      })
      expect(session.user.email).toBe('newuser@example.com')
    })
  })

  describe('signOut()', () => {
    it('should sign out and clear session', async () => {
      // Set up a session first
      const authResponse: AuthResponse = {
        access_token: 'token',
        refresh_token: 'refresh',
        expires_in: 3600,
        token_type: 'Bearer',
        user: { id: '1', email: 'user@example.com', created_at: '' },
      }

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse)
      await auth.signIn({ email: 'user@example.com', password: 'password' })

      // Now sign out
      vi.mocked(mockFetch.post).mockResolvedValue(undefined)
      await auth.signOut()

      expect(mockFetch.post).toHaveBeenCalledWith('/api/auth/signout')
      expect(auth.getSession()).toBeNull()
      expect(mockFetch.setAuthToken).toHaveBeenCalledWith(null)
      expect(localStorage.getItem('fluxbase.auth.session')).toBeNull()
    })

    it('should clear session even if API call fails', async () => {
      const authResponse: AuthResponse = {
        access_token: 'token',
        refresh_token: 'refresh',
        expires_in: 3600,
        token_type: 'Bearer',
        user: { id: '1', email: 'user@example.com', created_at: '' },
      }

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse)
      await auth.signIn({ email: 'user@example.com', password: 'password' })

      vi.mocked(mockFetch.post).mockRejectedValue(new Error('Network error'))

      await auth.signOut()

      expect(auth.getSession()).toBeNull()
    })
  })

  describe('refreshToken()', () => {
    it('should refresh access token', async () => {
      // Set up initial session
      const authResponse: AuthResponse = {
        access_token: 'old-token',
        refresh_token: 'refresh-token',
        expires_in: 3600,
        token_type: 'Bearer',
        user: { id: '1', email: 'user@example.com', created_at: '' },
      }

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse)
      await auth.signIn({ email: 'user@example.com', password: 'password' })

      // Refresh token
      const refreshResponse: AuthResponse = {
        access_token: 'new-token',
        refresh_token: 'new-refresh-token',
        expires_in: 3600,
        token_type: 'Bearer',
        user: { id: '1', email: 'user@example.com', created_at: '' },
      }

      vi.mocked(mockFetch.post).mockResolvedValue(refreshResponse)

      const session = await auth.refreshToken()

      expect(mockFetch.post).toHaveBeenCalledWith('/api/auth/refresh', {
        refresh_token: 'refresh-token',
      })
      expect(session.access_token).toBe('new-token')
      expect(mockFetch.setAuthToken).toHaveBeenCalledWith('new-token')
    })

    it('should throw error when no refresh token available', async () => {
      await expect(auth.refreshToken()).rejects.toThrow('No refresh token available')
    })
  })

  describe('getCurrentUser()', () => {
    it('should fetch current user', async () => {
      // Set up session
      const authResponse: AuthResponse = {
        access_token: 'token',
        refresh_token: 'refresh',
        expires_in: 3600,
        token_type: 'Bearer',
        user: { id: '1', email: 'user@example.com', created_at: '' },
      }

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse)
      await auth.signIn({ email: 'user@example.com', password: 'password' })

      const user = { id: '1', email: 'user@example.com', created_at: '' }
      vi.mocked(mockFetch.get).mockResolvedValue(user)

      const result = await auth.getCurrentUser()

      expect(mockFetch.get).toHaveBeenCalledWith('/api/auth/user')
      expect(result).toEqual(user)
    })

    it('should throw error when not authenticated', async () => {
      await expect(auth.getCurrentUser()).rejects.toThrow('Not authenticated')
    })
  })

  describe('updateUser()', () => {
    it('should update user profile', async () => {
      // Set up session
      const authResponse: AuthResponse = {
        access_token: 'token',
        refresh_token: 'refresh',
        expires_in: 3600,
        token_type: 'Bearer',
        user: { id: '1', email: 'old@example.com', created_at: '' },
      }

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse)
      await auth.signIn({ email: 'old@example.com', password: 'password' })

      const updatedUser = { id: '1', email: 'new@example.com', created_at: '' }
      vi.mocked(mockFetch.patch).mockResolvedValue(updatedUser)

      const result = await auth.updateUser({ email: 'new@example.com' })

      expect(mockFetch.patch).toHaveBeenCalledWith('/api/auth/user', {
        email: 'new@example.com',
      })
      expect(result.email).toBe('new@example.com')
      expect(auth.getUser()?.email).toBe('new@example.com')
    })

    it('should throw error when not authenticated', async () => {
      await expect(auth.updateUser({ email: 'new@example.com' })).rejects.toThrow(
        'Not authenticated'
      )
    })
  })

  describe('session persistence', () => {
    it('should not persist when persist is false', async () => {
      const noPersistAuth = new FluxbaseAuth(mockFetch, true, false)

      const authResponse: AuthResponse = {
        access_token: 'token',
        refresh_token: 'refresh',
        expires_in: 3600,
        token_type: 'Bearer',
        user: { id: '1', email: 'user@example.com', created_at: '' },
      }

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse)
      await noPersistAuth.signIn({ email: 'user@example.com', password: 'password' })

      expect(localStorage.getItem('fluxbase.auth.session')).toBeNull()
    })
  })
})
