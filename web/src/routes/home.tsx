import {type ReactNode, type SubmitEventHandler, useEffect, useState} from 'react'
import {useNavigate} from '@tanstack/react-router'
import {Pause, Play, Search, SkipBack, SkipForward} from 'lucide-react'
import {Button} from '#/components/ui/button'
import {Input} from '#/components/ui/input'
import {FastAverageColor} from 'fast-average-color'
import {Slider} from "#/components/ui/slider.tsx";
import type {PlaybackState, QueueItem} from '#/lib/types'


export function HeaderSearch({
                                 query,
                                 setQuery,
                                 onSubmitSearch,
                             }: {
    query: string
    setQuery: (value: string) => void
    onSubmitSearch: SubmitEventHandler<HTMLFormElement>
}) {
    return (
        <header
            className="z-20 border-b border-black/35 bg-[#0b0b10]/55 py-3 shadow-[0_10px_30px_rgba(0,0,0,0.35)] backdrop-blur-md">
            <form className="mx-auto w-full max-w-140" onSubmit={onSubmitSearch}>
                <div className="relative w-full">
                    <Search
                        className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"/>
                    <Input
                        value={query}
                        onChange={(event) => setQuery(event.target.value)}
                        placeholder="Search or paste YouTube URL"
                        className="h-11 rounded-full border-0 bg-mauve-800 pl-10 text-white placeholder:text-white/65 caret-white"
                    />
                </div>
            </form>
        </header>
    )
}

function HomeRouteLayout({
                             query,
                             setQuery,
                             children,
                         }: {
    query: string
    setQuery: (value: string) => void
    children: ReactNode
    withBottomDockPadding?: boolean
}) {
    const navigate = useNavigate()

    const submitHeaderSearch: SubmitEventHandler<HTMLFormElement> = (event) => {
        event.preventDefault()
        const q = query.trim()
        if (!q) return
        navigate({to: '/search', search: {q}})
    }

    return (
        <main
            className="mx-auto grid h-dvh w-full max-w-full grid-rows-[auto_minmax(0,1fr)] text-white"
        >
            <HeaderSearch
                query={query}
                setQuery={setQuery}
                onSubmitSearch={submitHeaderSearch}
            />
            <div className="site-main min-h-0 overflow-y-auto">
                {children}
            </div>
        </main>
    )
}

export default HomeRouteLayout

function isGoogleusercontentUrl(url: string) {
    try {
        const parsed = new URL(url, window.location.origin)
        return parsed.hostname.endsWith('googleusercontent.com')
    } catch {
        return false
    }
}

export function BottomNowPlayingDock({
                                         coverUrl,
                                         title,
                                         onOpenNowPlaying,
                                     }: {
    coverUrl: string
    title: string
    onOpenNowPlaying: () => void
}) {
    return (
        <div className="fixed inset-x-0 bottom-0 z-40 border-t border-white/10 bg-black/70 px-4 py-3 backdrop-blur">
            <div className="mx-auto flex w-full items-center gap-3">
                <div
                    role="button"
                    tabIndex={0}
                    onClick={onOpenNowPlaying}
                    onKeyDown={(event) => {
                        if (event.key === 'Enter' || event.key === ' ') {
                            event.preventDefault()
                            onOpenNowPlaying()
                        }
                    }}
                    className="flex min-w-0 flex-1 items-center gap-3 text-left"
                    aria-label="Open now playing"
                >
                    <div className="h-11 w-11 overflow-hidden rounded-md bg-white/10">
                        <img src={coverUrl} alt="" referrerPolicy="no-referrer"
                             className="h-full w-full object-cover opacity-90"/>
                    </div>
                    <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium">{title}</p>
                        <input
                            type="range"
                            min={0}
                            max={100}
                            value={35}
                            readOnly
                            className="mt-1 h-1 w-full accent-white/80"
                        />
                    </div>
                </div>
                <div className="flex items-center gap-1">
                    <Button variant="ghost" size="icon" aria-label="Previous">
                        <SkipBack className="h-4 w-4"/>
                    </Button>
                    <Button variant="ghost" size="icon" aria-label="Pause">
                        <Pause className="h-4 w-4"/>
                    </Button>
                    <Button variant="ghost" size="icon" aria-label="Next">
                        <SkipForward className="h-4 w-4"/>
                    </Button>
                </div>
            </div>
        </div>
    )
}

function formatClock(totalSeconds: number) {
    if (!Number.isFinite(totalSeconds) || totalSeconds <= 0) {
        return '0:00'
    }
    const rounded = Math.floor(totalSeconds)
    const minutes = Math.floor(rounded / 60)
    const seconds = rounded % 60
    return `${minutes}:${seconds.toString().padStart(2, '0')}`
}

function PlaybackControl({
                             playbackState,
                             positionSec,
                             durationSec,
                             isCommandPending,
                             onTogglePlayPause,
                             onSkip,
                             onRestart,
                             onSeek,
                         }: {
    playbackState: PlaybackState
    positionSec: number
    durationSec: number
    isCommandPending: boolean
    onTogglePlayPause: () => void
    onSkip: () => void
    onRestart: () => void
    onSeek: (positionSec: number) => void
}) {
    const hasKnownDuration = durationSec > 0
    const sliderMax = hasKnownDuration ? durationSec : Math.max(positionSec + 120, 300)
    const safePosition = Math.max(0, Math.min(positionSec, sliderMax))
    const canSeek = playbackState === 'playing' || playbackState === 'paused' || positionSec > 0
    const canRestart = canSeek
    const isPlaying = playbackState === 'playing'
    const canToggle = playbackState === 'playing' || playbackState === 'paused'
    const [sliderValue, setSliderValue] = useState(safePosition)
    const [isScrubbing, setIsScrubbing] = useState(false)

    useEffect(() => {
        if (!isScrubbing) {
            setSliderValue(safePosition)
        }
    }, [safePosition, isScrubbing])

    return (
        <div className="flex w-full flex-col items-center">
            <div className="flex items-center gap-8 flex-row">
                <Button variant="ghost" size="default"
                        className="rounded-full w-12 h-12 p-0 hover:bg-white/20"
                        onClick={onRestart}
                        disabled={isCommandPending || !canRestart}
                        aria-label="Restart track">
                    <SkipBack/>
                </Button>
                <Button variant="ghost" size="default"
                        className="rounded-full w-18 h-18 p-0 hover:bg-white/20"
                        onClick={onTogglePlayPause}
                        disabled={isCommandPending || !canToggle}
                        aria-label={isPlaying ? 'Pause' : 'Resume'}>
                    {isPlaying ? <Pause className="scale-150"/> : <Play className="scale-150"/>}
                </Button>

                <Button variant="ghost" size="default"
                        className="rounded-full w-12 h-12 p-0 hover:bg-white/20 "
                        onClick={onSkip}
                        disabled={isCommandPending}
                        aria-label="Skip track">
                    <SkipForward/>
                </Button>
            </div>

            <div className="w-full flex items-center flex-row gap-4">
                <span>{formatClock(safePosition)}</span>
                <Slider
                    value={[sliderValue]}
                    max={sliderMax}
                    min={0}
                    step={1}
                    disabled={isCommandPending || !canSeek}
                    onValueChange={(value) => {
                        setIsScrubbing(true)
                        setSliderValue(Math.max(0, Math.min(value[0] ?? 0, sliderMax)))
                    }}
                    onValueCommit={(value) => {
                        const nextPositionSec = Math.max(0, Math.min(value[0] ?? 0, sliderMax))
                        setSliderValue(nextPositionSec)
                        setIsScrubbing(false)
                        onSeek(nextPositionSec)
                    }}
                    className="**:data-[slot=slider-track]:bg-white/30 **:data-[slot=slider-range]:bg-white **:data-[slot=slider-thumb]:border-white **:data-[slot=slider-thumb]:bg-white"
                />
                <span>{hasKnownDuration ? formatClock(durationSec) : '--:--'}</span>
            </div>
        </div>
    );
}

export function Hero({
                         src,
                         title,
                         subtitle,
                         playbackState = 'stopped',
                         positionSec = 0,
                         durationSec = 0,
                         isCommandPending = false,
                         onTogglePlayPause,
                         onSkip,
                         onRestart,
                         onSeek,
                         queueItems = [],
                         queueIsLoading = false,
                     }: {
    src: string
    title: string
    subtitle?: string
    playbackState?: PlaybackState
    positionSec?: number
    durationSec?: number
    isCommandPending?: boolean
    onTogglePlayPause?: () => void
    onSkip?: () => void
    onRestart?: () => void
    onSeek?: (positionSec: number) => void
    queueItems?: QueueItem[]
    queueIsLoading?: boolean
}) {
    const [bgColor, setBgColor] = useState('rgb(20,20,20)')
    const [bgColorMid, setBgColorMid] = useState('rgb(10,10,10)')

    function rgbToHsl(r: number, g: number, b: number) {
        const rn = r / 255
        const gn = g / 255
        const bn = b / 255
        const max = Math.max(rn, gn, bn)
        const min = Math.min(rn, gn, bn)
        const delta = max - min

        let h = 0
        let s = 0
        const l = (max + min) / 2

        if (delta !== 0) {
            s = delta / (1 - Math.abs(2 * l - 1))
            switch (max) {
                case rn:
                    h = ((gn - bn) / delta) % 6
                    break
                case gn:
                    h = (bn - rn) / delta + 2
                    break
                default:
                    h = (rn - gn) / delta + 4
            }
            h *= 60
            if (h < 0) h += 360
        }

        return {h, s, l}
    }

    function hslToRgb(h: number, s: number, l: number) {
        const c = (1 - Math.abs(2 * l - 1)) * s
        const x = c * (1 - Math.abs(((h / 60) % 2) - 1))
        const m = l - c / 2

        let r1 = 0
        let g1 = 0
        let b1 = 0

        if (h < 60) {
            r1 = c
            g1 = x
        } else if (h < 120) {
            r1 = x
            g1 = c
        } else if (h < 180) {
            g1 = c
            b1 = x
        } else if (h < 240) {
            g1 = x
            b1 = c
        } else if (h < 300) {
            r1 = x
            b1 = c
        } else {
            r1 = c
            b1 = x
        }

        return {
            r: Math.round((r1 + m) * 255),
            g: Math.round((g1 + m) * 255),
            b: Math.round((b1 + m) * 255),
        }
    }

    useEffect(() => {
        if (isGoogleusercontentUrl(src)) {
            setBgColor('rgb(20,20,20)')
            setBgColorMid('rgb(10,10,10)')
            return
        }

        const fac = new FastAverageColor()
        let cancelled = false

        fac
            .getColorAsync(src, { crossOrigin: 'anonymous'})
            .then((color) => {
                if (cancelled) return
                const [r, g, b] = color.value
                const {h, s, l} = rgbToHsl(r, g, b)
                const adjusted = hslToRgb(h, Math.min(1, Math.max(0.41, s * 0.91)), Math.max(0.4, l * 0.95))
                const muted = `rgb(${adjusted.r}, ${adjusted.g}, ${adjusted.b})`
                const midpoint = `rgb(${Math.round(adjusted.r * 0.5)}, ${Math.round(adjusted.g * 0.5)}, ${Math.round(adjusted.b * 0.5)})`
                setBgColor(muted)
                setBgColorMid(midpoint)
            })
            .catch(() => {
                if (!cancelled) {
                    setBgColor('rgb(20,20,20)')
                    setBgColorMid('rgb(10,10,10)')
                }
            })

        return () => {
            cancelled = true
            fac.destroy()
        }
    }, [src])

    return (
        <>
            <section className="relative h-full min-h-full w-full overflow-hidden transition-colors duration-700">
                <div className="pointer-events-none absolute inset-0" style={{backgroundColor: bgColor}}/>
                <div
                    className="pointer-events-none absolute inset-x-0 top-0 h-24 bg-linear-to-b from-black/35 to-transparent"/>
                <div
                    className="pointer-events-none absolute inset-x-0 bottom-0 h-48"
                    style={{backgroundImage: `linear-gradient(to bottom, rgba(0,0,0,0) 0%, ${bgColorMid} 100%)`}}
                />
                <div
                    className="pointer-events-none absolute -left-20 top-1/2 h-64 w-64 -translate-y-1/2 rounded-full blur-[120px]"
                    style={{backgroundColor: bgColor, opacity: 0.9}}
                />
                <div
                    className="pointer-events-none absolute -right-20 top-1/2 h-64 w-64 -translate-y-1/2 rounded-full blur-[120px]"
                    style={{backgroundColor: bgColor, opacity: 0.85}}
                />

                <div
                    className="relative mx-auto flex h-full min-h-0 w-full flex-col items-center gap-6 px-6 py-8 text-center">
                    <div className="relative flex min-h-0 w-full flex-1 items-center justify-center">
                        <img
                            src={src}
                            alt={title}
                            referrerPolicy="no-referrer"
                            className="aspect-square h-full w-auto max-h-full max-w-full rounded-[1.75rem] object-cover shadow-[0_30px_80px_rgba(0,0,0,0.6)]"
                        />
                    </div>


                    <h1 className="shrink-0 text-3xl font-semibold tracking-tight text-white sm:text-4xl">{title}</h1>
                    {subtitle ? <p className="mt-3 shrink-0 text-sm text-white/75 sm:text-base">{subtitle}</p> : null}

                    <PlaybackControl
                        playbackState={playbackState}
                        positionSec={positionSec}
                        durationSec={durationSec}
                        isCommandPending={isCommandPending}
                        onTogglePlayPause={onTogglePlayPause ?? (() => {
                        })}
                        onSkip={onSkip ?? (() => {
                        })}
                        onRestart={onRestart ?? (() => {
                        })}
                        onSeek={onSeek ?? (() => {
                        })}
                    />
                </div>
            </section>

            <section
                className="w-full bg-black"
                style={{backgroundImage: `linear-gradient(to bottom, ${bgColorMid} 0%, rgb(0,0,0) 68%)`}}
            >
                <div className="mx-auto w-full px-6 pb-10 pt-16">
                    <div className="rounded-3xl border border-white/12 bg-white/6 p-5 backdrop-blur-sm">
                        <div className="mb-4 flex items-center justify-between">
                            <h2 className="text-lg font-semibold tracking-tight text-white">Up Next</h2>
                            <span className="text-xs text-white/65">{queueItems.length} queued</span>
                        </div>

                        {queueIsLoading ? (
                            <p className="text-sm text-white/70">Loading queue...</p>
                        ) : queueItems.length === 0 ? (
                            <p className="text-sm text-white/70">Queue is empty. Add tracks from Search.</p>
                        ) : (
                            <ul className="space-y-2">
                                {queueItems.slice(0, 25).map((item, index) => (
                                    <li
                                        key={item.id ?? `${item.title ?? item.source ?? 'track'}-${index}`}
                                        className="flex items-center gap-3 rounded-xl border border-white/10 bg-black/25 px-3 py-2"
                                    >
                                        <span
                                            className="w-6 shrink-0 text-right text-xs text-white/55">{index + 1}</span>
                                        <div className="h-10 w-10 shrink-0 overflow-hidden rounded-md bg-white/8">
                                            {item.coverUrl ? (
                                                <img src={item.coverUrl} alt="" referrerPolicy="no-referrer"
                                                     className="h-full w-full object-cover"/>
                                            ) : null}
                                        </div>
                                        <div className="min-w-0 flex-1">
                                            <p className="truncate text-sm font-medium text-white">
                                                {item.title ?? item.id ?? 'unknown track'}
                                            </p>
                                            {item.source ? (
                                                <p className="truncate text-xs text-white/60">{item.source}</p>
                                            ) : null}
                                        </div>
                                    </li>
                                ))}
                            </ul>
                        )}
                    </div>
                </div>
            </section>
        </>
    )
}
