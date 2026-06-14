import { createTheme } from '@mui/material/styles'

export const theme = createTheme({
  palette: {
    mode: 'light',
    primary: { main: '#2454ff' },
    background: { default: '#f6f8fb' },
  },
  shape: { borderRadius: 10 },
  components: {
    MuiTextField: {
      defaultProps: { size: 'small' },
    },
    MuiFormControl: {
      defaultProps: { size: 'small' },
    },
    MuiButton: {
      defaultProps: { size: 'small' },
    },
    MuiIconButton: {
      defaultProps: { size: 'small' },
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
