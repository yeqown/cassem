import React from 'react'
import ReactDOM from 'react-dom/client'
import { CssBaseline, ThemeProvider } from '@mui/material'
import { RouterProvider, createBrowserRouter } from 'react-router-dom'
import { routes } from './routes'
import { resolveRouterBasename } from './lib/routerBase'
import { theme } from './theme'

const router = createBrowserRouter(routes, { basename: resolveRouterBasename(import.meta.env.BASE_URL) })

ReactDOM.createRoot(document.getElementById('cassem-admin')!).render(
  <React.StrictMode>
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <RouterProvider router={router} />
    </ThemeProvider>
  </React.StrictMode>,
)
