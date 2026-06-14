import { Alert, Box, CircularProgress, Typography } from '@mui/material'

export function LoadingState({ label = 'Loading' }: { label?: string }) {
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, p: 3 }}>
      <CircularProgress size={20} />
      <Typography>{label}</Typography>
    </Box>
  )
}

export function ErrorState({ message }: { message: string }) {
  return (
    <Alert severity="error" sx={{ my: 2 }}>
      {message}
    </Alert>
  )
}

export function EmptyState({ title, description }: { title: string; description?: string }) {
  return (
    <Box sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
      <Typography variant="h6">{title}</Typography>
      {description && <Typography>{description}</Typography>}
    </Box>
  )
}
