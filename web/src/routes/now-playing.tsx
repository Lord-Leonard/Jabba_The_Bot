import HomeRouteLayout, {Hero} from './home'
import usePlayerRuntime from './home-data'

export function NowPlayingPage() {
  const player = usePlayerRuntime()
  const isPlayingSomething = Boolean(player.nowPlaying.data?.title?.trim())
  const playbackState = player.state.data?.playbackState ?? 'stopped'
  const positionSec = player.nowPlaying.data?.position ?? 0
  const durationSec = player.nowPlaying.data?.duration ?? 0

  const togglePlayPause = () => {
    if (playbackState === 'playing') {
      player.run('pause')
      return
    }
    if (playbackState === 'paused') {
      player.run('resume')
    }
  }

  return (
    <HomeRouteLayout query={player.query} setQuery={player.setQuery} withBottomDockPadding={false}>
      <Hero
        src={player.coverUrl}
        title={player.nowPlaying.data?.title ?? 'Nothing currently playing'}
        subtitle={isPlayingSomething ? 'Artist • Album' : 'Start a track from search or discover what is hot right now.'}
        playbackState={playbackState}
        positionSec={positionSec}
        durationSec={durationSec}
        isCommandPending={player.command.isPending}
        onTogglePlayPause={togglePlayPause}
        onSkip={() => player.run('skip')}
        onRestart={() => player.run('seek', {positionSec: 0})}
        onSeek={(nextPositionSec) => player.run('seek', {positionSec: Math.max(0, Math.floor(nextPositionSec))})}
        queueItems={player.queue.data ?? []}
        queueIsLoading={player.queue.isLoading}
      />
    </HomeRouteLayout>
  )
}
