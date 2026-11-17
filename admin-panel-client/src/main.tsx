import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider, createRouter } from '@tanstack/react-router'
import { QueryCache, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { routeTree } from '@/routeTree.gen'
import { isApiError } from '@/lib/api'
import { meQueryKey } from '@/lib/auth'

import './index.css'
import { createTheme, CssBaseline, ThemeProvider } from '@mui/material';

const queryClient = new QueryClient({
  queryCache: new QueryCache({
    // A 401 from any query means the session expired or was revoked: go back to /login.
    onError: (error) => {
      if (!isApiError(error) || error.errorCode !== 401) return
      queryClient.setQueryData(meQueryKey, null)
      const { pathname, searchStr } = router.state.location
      if (pathname !== '/login') {
        router.navigate({ to: '/login', search: { redirect: pathname + searchStr }, replace: true })
      }
    },
  }),
})

const router = createRouter({
  routeTree,
  context: { queryClient },
  defaultPreload: 'intent',
})

// This is what makes Link/navigate fully type-aware across your app:
declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

const theme = createTheme(
  {
    palette: { mode: 'light' },
    typography: {
      fontFamily: '"Inter", "Helvetica", "Arial", sans-serif',
      h1: { fontWeight: 700, letterSpacing: '-0.02em' },
      h2: { fontWeight: 700, letterSpacing: '-0.015em' },
      h3: { fontWeight: 600, letterSpacing: '-0.01em' },
      h4: { fontWeight: 600 },
      h5: { fontWeight: 600 },
      h6: { fontWeight: 600 },
      subtitle1: { fontWeight: 500 },
      subtitle2: { fontWeight: 500 },
      body1: { fontWeight: 400 },
      body2: { fontWeight: 400 },
      button: { fontWeight: 600, textTransform: 'none' },
      caption: { fontWeight: 500, letterSpacing: '0.02em' },
      overline: { fontWeight: 600, letterSpacing: '0.08em' },
    },
  }
)

async function enableMocking() {
  if (!import.meta.env.DEV) return
  const { worker } = await import('./mocks/browser')
  return worker.start({ onUnhandledFrame: 'warn' })
}

enableMocking().then(() => {
  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <QueryClientProvider client={queryClient}>
        <ThemeProvider theme={theme}>
          <CssBaseline />
          <RouterProvider router={router} />
        </ThemeProvider>
      </QueryClientProvider>
    </StrictMode>,
  )
})
