import {useNavigate} from '@tanstack/react-router'
import {Card, CardContent, CardHeader, CardTitle} from '#/components/ui/card'
import HomeRouteLayout, {BottomNowPlayingDock} from './home'
import usePlayerRuntime, {hotSections} from './home-data'

export function LandingPage() {
  const navigate = useNavigate()
  const home = usePlayerRuntime()

  return (
    <HomeRouteLayout query={home.query} setQuery={home.setQuery}>
      <section className="grid gap-4 md:grid-cols-3">
        {hotSections.map((section) => {
          const Icon = section.icon
          return (
            <Card key={section.title}>
              <CardHeader className="flex flex-row items-center justify-between space-y-0">
                <CardTitle>{section.title}</CardTitle>
                <Icon className="h-4 w-4 text-muted-foreground"/>
              </CardHeader>
              <CardContent>
                <ul className="space-y-2 text-sm">
                  {section.items.map((item) => (
                    <li key={item} className="rounded-md border border-white/10 px-3 py-2">
                      {item}
                    </li>
                  ))}
                </ul>
              </CardContent>
            </Card>
          )
        })}
      </section>

      <BottomNowPlayingDock
        coverUrl={home.coverUrl}
        title={home.nowPlaying.data?.title ?? 'Nothing currently playing'}
        onOpenNowPlaying={() => navigate({to: '/'})}
      />

      {(home.state.error || home.nowPlaying.error || home.queue.error) && (
        <p className="text-sm text-destructive">
          API unavailable. Start your Go API on <code>localhost:8080</code> and check `/api/state`,
          `/api/queue`, `/api/now-playing`, `/api/events`.
        </p>
      )}

      {home.isLoading && <p className="text-sm text-muted-foreground">Loading current bot state...</p>}
    </HomeRouteLayout>
  )
}
