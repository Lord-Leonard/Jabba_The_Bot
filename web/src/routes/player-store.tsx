import {type ReactNode, useMemo, useState} from 'react'
import {PlayerStoreContext, type PlayerStoreValue} from './player-store-context'

export function PlayerStoreProvider({children}: {children: ReactNode}) {
  const [query, setQuery] = useState('')
  const [error, setError] = useState<string | null>(null)

  const value = useMemo<PlayerStoreValue>(
    () => ({
      query,
      setQuery,
      error,
      setError,
      coverUrl: 'https://i.ytimg.com/vi/I8XwGtAwUgE/maxresdefault.jpg',
    }),
    [error, query],
  )

  return <PlayerStoreContext.Provider value={value}>{children}</PlayerStoreContext.Provider>
}
