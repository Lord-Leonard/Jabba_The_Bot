import {useEffect, useMemo} from 'react'
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query'
import {Disc3, Flame, Sparkles} from 'lucide-react'
import {getNowPlaying, getQueue, getState, randomId, sendCommand} from '#/lib/api'
import type {CommandRequest} from '#/lib/types'
import {usePlayerStore} from './player-store-context'

const queryKeys = {
  state: ['state'] as const,
  nowPlaying: ['now-playing'] as const,
  queue: ['queue'] as const,
}

export const hotSections = [
  {
    title: 'Album Charts',
    icon: Disc3,
    items: ['Sabrina Carpenter - Short n Sweet', 'Billie Eilish - Hit Me Hard and Soft', 'Taylor Swift - The Tortured Poets Department'],
  },
  {
    title: 'New Releases',
    icon: Sparkles,
    items: ['Rosalia - Motomami Sessions', 'Fred again.. - USB Mix 11', 'Doja Cat - Scarlet 2: Electric'],
  },
  {
    title: 'Singles Charts',
    icon: Flame,
    items: ['Chappell Roan - Good Luck, Babe!', 'Teddy Swims - Lose Control', 'Artemas - i like the way you kiss me'],
  },
]

function useEventSync() {
  const queryClient = useQueryClient()

  useEffect(() => {
    const source = new EventSource('/api/events')

    const invalidate = () => {
      queryClient.invalidateQueries({queryKey: queryKeys.state})
      queryClient.invalidateQueries({queryKey: queryKeys.nowPlaying})
      queryClient.invalidateQueries({queryKey: queryKeys.queue})
    }

    source.onmessage = invalidate
    source.addEventListener('snapshot', invalidate)
    source.addEventListener('playback_state_changed', invalidate)
    source.addEventListener('queue_updated', invalidate)
    source.addEventListener('now_playing', invalidate)
    source.addEventListener('progress', invalidate)

    return () => source.close()
  }, [queryClient])
}

function useCommand() {
  return useMutation({
    mutationFn: (command: CommandRequest) =>
      sendCommand({
        ...command,
        clientRequestId: randomId(),
      }),
  })
}

function usePlayerRuntime(initialQuery = '') {
  useEventSync()

  const {query, setQuery, error, setError, coverUrl} = usePlayerStore()

  const state = useQuery({queryKey: queryKeys.state, queryFn: getState, refetchInterval: 10000})
  const nowPlaying = useQuery({queryKey: queryKeys.nowPlaying, queryFn: getNowPlaying, refetchInterval: 10000})
  const queue = useQuery({queryKey: queryKeys.queue, queryFn: getQueue, refetchInterval: 10000})

  const command = useCommand()

  useEffect(() => {
    setQuery(initialQuery)
  }, [initialQuery, setQuery])

  async function run(type: string, payload?: Record<string, unknown>) {
    setError(null)
    try {
      const result = await command.mutateAsync({type, payload})
      if (!result.accepted) {
        setError(result.error?.message ?? 'Command rejected by server')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Request failed')
    }
  }

  function enqueueQuery() {
    const trimmed = query.trim()
    if (!trimmed) return
    run('enqueue', {query: trimmed})
    setQuery('')
  }

  const isLoading = useMemo(
    () => state.isLoading || nowPlaying.isLoading || queue.isLoading,
    [state.isLoading, nowPlaying.isLoading, queue.isLoading],
  )

  const filteredQueue = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase()
    if (!normalizedQuery || !queue.data) return []
    return queue.data.filter((item) => (item.title ?? item.id ?? '').toLowerCase().includes(normalizedQuery))
  }, [queue.data, query])

  const resolvedCoverUrl = nowPlaying.data?.coverUrl ?? coverUrl

  return {
    query,
    setQuery,
    error,
    state,
    nowPlaying,
    queue,
    command,
    run,
    enqueueQuery,
    isLoading,
    filteredQueue,
    coverUrl: resolvedCoverUrl,
    // coverUrl: 'https://i.ytimg.com/vi/TkiyWhETiJw/maxresdefault.jpg',
    // coverUrl: 'https://i.ytimg.com/vi/dQw4w9WgXcQ/hq720.jpg',

  }
}

export default usePlayerRuntime
