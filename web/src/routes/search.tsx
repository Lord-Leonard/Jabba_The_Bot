import {useQuery} from '@tanstack/react-query'
import {useNavigate, useSearch} from '@tanstack/react-router'
import {search as searchApi} from '#/lib/api'
import HomeRouteLayout, {BottomNowPlayingDock} from "./home"
import usePlayerRuntime from "#/routes/home-data.ts";

export function SearchPage() {
    const navigate = useNavigate()

    const player = usePlayerRuntime()
    const routeSearch = useSearch({from: '/search'})
    const query = routeSearch.q.trim()
    const resultsQuery = useQuery({
        queryKey: ['search', query],
        queryFn: () => searchApi(query),
        enabled: query.length > 0,
    })

    const items = resultsQuery.data ?? []
    return (

        <HomeRouteLayout query={player.query} setQuery={player.setQuery} withBottomDockPadding={false}>
            <section
                className="w-full bg-black"
            >
                <div className="mx-auto w-full px-6 pb-10 pt-16">
                    <div className="rounded-3xl border border-white/12 bg-white/6 p-5 backdrop-blur-sm">

                        {resultsQuery.isLoading ? (
                            <p className="text-sm text-white/70">Loading queue...</p>
                        ) : items.length === 0 ? (
                            <p className="text-sm text-white/70">Queue is empty. Add tracks from Search.</p>
                        ) : (
                            <ul className="space-y-2">
                                {items.slice(0, 25).map((item, index) => (
                                    <li
                                        key={item.videoId ?? `${item.title}-${index}`}
                                        className="flex items-center gap-3 rounded-xl border border-white/10 bg-black/25 px-3 py-2"
                                    >
                                        <span
                                            className="w-6 shrink-0 text-right text-xs text-white/55">{index + 1}</span>
                                        <div className="h-10 w-10 shrink-0 overflow-hidden rounded-md bg-white/8">
                                            {item.coverArtUrl ? (
                                                <img src={item.coverArtUrl} alt=""
                                                     referrerPolicy="no-referrer"
                                                     className="h-full w-full object-cover"/>
                                            ) : null}
                                        </div>
                                        <div className="min-w-0 flex-1">
                                            <p className="truncate text-sm font-medium text-white">
                                                {item.title ?? item.videoId ?? 'unknown track'}
                                            </p>
                                        </div>
                                    </li>
                                ))}
                            </ul>
                        )}
                    </div>
                </div>
            </section>

            <BottomNowPlayingDock
                coverUrl={player.coverUrl}
                title={player.nowPlaying.data?.title ?? 'Nothing currently playing'}
                onOpenNowPlaying={() => navigate({to: '/'})}
            />
        </HomeRouteLayout>
    )
}
