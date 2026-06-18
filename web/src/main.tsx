import React from 'react'
import ReactDOM from 'react-dom/client'
import { CssBaseline } from '@mui/material'
import { RouterProvider, createBrowserRouter } from 'react-router-dom'
import { AppThemeProvider } from './AppThemeProvider'
import { ToastProvider } from './components/ToastProvider'
import { routes } from './routes'
import { resolveRouterBasename } from './lib/routerBase'

const router = createBrowserRouter(routes, { basename: resolveRouterBasename(import.meta.env.BASE_URL) })

ReactDOM.createRoot(document.getElementById('cassem-admin')!).render(
  <React.StrictMode>
    <AppThemeProvider>
      <ToastProvider>
        <CssBaseline />
        <RouterProvider router={router} />
      </ToastProvider>
    </AppThemeProvider>
  </React.StrictMode>,
)
