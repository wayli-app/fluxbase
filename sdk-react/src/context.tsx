/**
 * React context for Fluxbase client
 */

import { createContext, useContext, type ReactNode } from 'react'
import type { FluxbaseClient } from '@fluxbase/sdk'

const FluxbaseContext = createContext<FluxbaseClient | null>(null)

export interface FluxbaseProviderProps {
  client: FluxbaseClient
  children: ReactNode
}

/**
 * Provider component to make Fluxbase client available throughout the app
 */
export function FluxbaseProvider({ client, children }: FluxbaseProviderProps) {
  return <FluxbaseContext.Provider value={client}>{children}</FluxbaseContext.Provider>
}

/**
 * Hook to access the Fluxbase client from context
 */
export function useFluxbaseClient(): FluxbaseClient {
  const client = useContext(FluxbaseContext)

  if (!client) {
    throw new Error('useFluxbaseClient must be used within a FluxbaseProvider')
  }

  return client
}
