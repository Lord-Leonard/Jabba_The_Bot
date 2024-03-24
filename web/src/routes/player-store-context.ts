import {createContext, useContext} from 'react'

export interface PlayerStoreValue {
  query: string
  setQuery: (value: string) => void
  error: string | null
  setError: (value: string | null) => void
  coverUrl: string
}

export const PlayerStoreContext = createContext<PlayerStoreValue | null>(null)

export function usePlayerStore() {
  const context = useContext(PlayerStoreContext)
  if (!context) {
    throw new Error('usePlayerStore must be used inside a PlayerStoreProvider')
  }
  return context
}
