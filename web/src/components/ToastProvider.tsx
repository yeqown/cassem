import CloseIcon from '@mui/icons-material/Close'
import { Alert, IconButton, Snackbar } from '@mui/material'
import { createContext, useCallback, useContext, useMemo, useRef, useState, type ReactElement, type ReactNode } from 'react'

export type ToastSeverity = 'success' | 'error' | 'warning' | 'info'

type ToastState = {
  id: number
  message: string
  severity: ToastSeverity
}

type ToastContextValue = {
  showToast: (message: string, severity?: ToastSeverity) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

export function ToastProvider({ children }: { children: ReactNode }): ReactElement {
  const [toast, setToast] = useState<ToastState | null>(null)
  const toastIdRef = useRef(0)

  const closeToast = useCallback((_event?: unknown, reason?: string) => {
    if (reason === 'clickaway') return
    setToast(null)
  }, [])

  const showToast = useCallback((message: string, severity: ToastSeverity = 'info') => {
    const trimmedMessage = message.trim()
    if (!trimmedMessage) return

    toastIdRef.current += 1
    setToast({ id: toastIdRef.current, message: trimmedMessage, severity })
  }, [])

  const value = useMemo<ToastContextValue>(() => ({ showToast }), [showToast])

  return (
    <ToastContext.Provider value={value}>
      {children}
      <Snackbar
        key={toast?.id}
        open={Boolean(toast)}
        autoHideDuration={4000}
        onClose={closeToast}
        anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
      >
        <Alert
          severity={toast?.severity || 'info'}
          variant="standard"
          onClose={closeToast}
          action={
            <IconButton color="inherit" size="small" aria-label="Close toast" onClick={() => closeToast()}>
              <CloseIcon fontSize="small" />
            </IconButton>
          }
          sx={{ width: '100%', borderRadius: 0, alignItems: 'center' }}
        >
          {toast?.message}
        </Alert>
      </Snackbar>
    </ToastContext.Provider>
  )
}

export function useToast() {
  const context = useContext(ToastContext)
  if (!context) {
    throw new Error('useToast must be used within ToastProvider')
  }

  return context
}
