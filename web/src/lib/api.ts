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

export function getState() {
  return request<BotState>('/api/state')
}

export function getQueue() {
  return request<QueueItem[]>('/api/queue')
}

export function getNowPlaying() {
  return request<NowPlaying | null>('/api/now-playing')
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
