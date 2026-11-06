import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { ThemeProvider } from '@mui/material/styles'
import { createAppTheme } from './theme'
import { readSettings, settingsChangedEvent } from './lib/settings'
import { I18nProvider, useI18n } from './i18n'

export function AppThemeProvider({ children }: { children: ReactNode }) {
  return <I18nProvider><LocalizedThemeProvider>{children}</LocalizedThemeProvider></I18nProvider>
}

function LocalizedThemeProvider({ children }: { children: ReactNode }) {
  const { locale } = useI18n()
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

  const theme = useMemo(() => createAppTheme(uiTheme, locale), [uiTheme, locale])

  return <ThemeProvider theme={theme}>{children}</ThemeProvider>
}
