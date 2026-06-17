import { createTheme } from '@mui/material/styles'
import { defaultSettings, getUIThemeOption, type UITheme } from './lib/settings'

export function createAppTheme(uiTheme: UITheme = defaultSettings.uiTheme) {
  const palette = getUIThemeOption(uiTheme)

  return createTheme({
    palette: {
      mode: 'light',
      primary: { main: palette.primary },
      secondary: { main: palette.secondary },
      background: { default: '#f6f8fb' },
    },
    shape: { borderRadius: 0 },
    components: {
      MuiTextField: {
        defaultProps: { size: 'small' },
      },
      MuiFormControl: {
        defaultProps: { size: 'small' },
      },
      MuiButton: {
        defaultProps: { size: 'small' },
        styleOverrides: {
          root: {
            borderRadius: 0,
          },
        },
      },
      MuiChip: {
        styleOverrides: {
          root: {
            borderRadius: 0,
          },
        },
      },
      MuiPaper: {
        styleOverrides: {
          root: {
            borderRadius: 0,
          },
        },
      },
      MuiCard: {
        styleOverrides: {
          root: {
            borderRadius: 0,
          },
        },
      },
      MuiIconButton: {
        defaultProps: { size: 'small' },
        styleOverrides: {
          root: {
            borderRadius: 0,
          },
        },
      },
      MuiAlert: {
        styleOverrides: {
          root: {
            borderRadius: 0,
          },
        },
      },
      MuiTooltip: {
        styleOverrides: {
          tooltip: {
            borderRadius: 0,
          },
        },
      },
      MuiAutocomplete: {
        styleOverrides: {
          tag: {
            borderRadius: 0,
          },
          paper: {
            borderRadius: 0,
          },
        },
      },
      MuiTableCell: {
        styleOverrides: {
          root: {
            paddingTop: 10,
            paddingBottom: 10,
          },
        },
      },
      MuiDialogActions: {
        styleOverrides: {
          root: {
            paddingTop: 12,
            paddingBottom: 12,
            paddingLeft: 20,
            paddingRight: 20,
            gap: 8,
          },
        },
      },
    },
  })
}

export const theme = createAppTheme()
