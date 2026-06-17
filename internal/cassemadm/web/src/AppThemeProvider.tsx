import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { ThemeProvider } from '@mui/material/styles'
import { createAppTheme } from './theme'
import { readSettings, settingsChangedEvent } from './lib/settings'

export function AppThemeProvider({ children }: { children: ReactNode }) {
  const [uiTheme, setUITheme] = useState(() => readSettings().uiTheme)

  useEffect(() => {
    function syncTheme() {
      setUITheme(readSettings().uiTheme)
    }

    window.addEventListener(settingsChangedEvent, syncTheme)
    window.addEventListener('storage', syncTheme)
    return () => {
      window.removeEventListener(settingsChangedEvent, syncTheme)
      window.removeEventListener('storage', syncTheme)
    }
  }, [])

  const theme = useMemo(() => createAppTheme(uiTheme), [uiTheme])

  return <ThemeProvider theme={theme}>{children}</ThemeProvider>
}
