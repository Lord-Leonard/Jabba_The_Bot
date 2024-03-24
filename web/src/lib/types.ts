export type PlaybackState = 'stopped' | 'loading' | 'playing' | 'paused' | string

export interface BotState {
  playbackState?: PlaybackState
  isConnected?: boolean
  volume?: number
}

export interface QueueItem {
  id?: string
  title?: string
  source?: string
  coverUrl?: string
  duration?: number
  requestedBy?: string
}

export interface NowPlaying {
  id?: string
  title?: string
  source?: string
  coverUrl?: string
  position?: number
  duration?: number
}

export interface CommandRequest {
  type: string
  payload?: Record<string, unknown>
  clientRequestId?: string
}

export interface CommandResponse {
  accepted: boolean
  commandId?: string
  error?: {
    code?: string
    message?: string
  }
}

export interface SearchResult {
  title?: string
  artist?: string
  videoId?: string
  coverArtUrl?: string
}
