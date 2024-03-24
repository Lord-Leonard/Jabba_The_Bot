import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from '@tanstack/react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { router } from './router'
import { PlayerStoreProvider } from './routes/player-store'
import './styles.css'

const queryClient = new QueryClient()

createRoot(document.getElementById('app')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <PlayerStoreProvider>
        <RouterProvider router={router} />
      </PlayerStoreProvider>
    </QueryClientProvider>
  </StrictMode>,
)
