import type { BotState, CommandRequest, CommandResponse, NowPlaying, QueueItem, SearchResult } from './types'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })

  if (!response.ok) {
    const body = await response.text()
    throw new Error(body || `${response.status} ${response.statusText}`)
  }

  if (response.status === 204) {
    return undefined as T
  }

  return (await response.json()) as T
}

type NowPlayingApiResponse = {
  id?: string
  title?: string
  source?: string
  coverUrl?: string
  position?: number
  duration?: number
  videoId?: string
  Title?: string
  VideoID?: string
  URL?: string
  Position?: number
  Duration?: number
  CoverArtURL?: string
}

function toSeconds(value: number | undefined): number | undefined {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    return undefined
  }
  // Go time.Duration is encoded as nanoseconds in JSON.
  if (value > 1_000_000_000) {
    return Math.floor(value / 1_000_000_000)
  }
  return value
}

function normalizeNowPlaying(payload: NowPlayingApiResponse | null): NowPlaying | null {
  if (!payload) {
    return null
  }

  const videoId = payload.videoId ?? payload.VideoID
  const title = payload.title ?? payload.Title
  const source = payload.source ?? payload.URL

  return {
    id: payload.id ?? videoId,
    title,
    source,
    coverUrl: payload.coverUrl ?? payload.CoverArtURL ?? (videoId ? `https://i.ytimg.com/vi/${videoId}/hqdefault.jpg` : undefined),
    position: toSeconds(payload.position ?? payload.Position),
    duration: toSeconds(payload.duration ?? payload.Duration),
  }
}

export function getState() {
  return request<BotState>('/api/state')
}

export function getQueue() {
  return request<QueueItem[]>('/api/queue')
}

export function getNowPlaying() {
  return request<NowPlayingApiResponse | null>('/api/now-playing').then(normalizeNowPlaying)
}

export function search(q: string, limit = 10) {
  const params = new URLSearchParams({ q, limit: String(limit) })
  return request<SearchResult[]>(`/api/search?${params.toString()}`)
}

export function sendCommand(command: CommandRequest) {
  return request<CommandResponse>('/api/commands', {
    method: 'POST',
    body: JSON.stringify(command),
  })
}

export function randomId() {
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`
}
