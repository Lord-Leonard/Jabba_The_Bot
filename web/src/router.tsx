import { Outlet, createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import { LandingPage } from './routes/landing'
import { NowPlayingPage } from './routes/now-playing'
import { SearchPage } from './routes/search'

const rootRoute = createRootRoute({
  component: () => <Outlet />,
})

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: NowPlayingPage,
})

const discoverRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/discover',
  component: LandingPage,
})

const nowPlayingRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/now-playing',
  component: NowPlayingPage,
})

const searchRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/search',
  validateSearch: (search: Record<string, unknown>) => ({
    q: typeof search.q === 'string' ? search.q : '',
  }),
  component: SearchPage,
})

const routeTree = rootRoute.addChildren([indexRoute, discoverRoute, nowPlayingRoute, searchRoute])

export const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
