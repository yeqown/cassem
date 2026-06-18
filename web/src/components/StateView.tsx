import { useEffect } from 'react'
import { Box, CircularProgress, Typography } from '@mui/material'
import { useToast } from './ToastProvider'

const recentlyEmittedMessages = new Set<string>()

export function LoadingState({ label = 'Loading' }: { label?: string }) {
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, p: 3 }}>
      <CircularProgress size={20} />
      <Typography>{label}</Typography>
    </Box>
  )
}

export function ErrorState({ message }: { message: string }) {
  const { showToast } = useToast()
  const normalizedMessage = message.trim()

  useEffect(() => {
    if (!normalizedMessage || recentlyEmittedMessages.has(normalizedMessage)) return

    recentlyEmittedMessages.add(normalizedMessage)
    showToast(normalizedMessage, 'error')

    setTimeout(() => {
      recentlyEmittedMessages.delete(normalizedMessage)
    }, 0)
  }, [normalizedMessage, showToast])

  return null
}

export function EmptyState({ title, description }: { title: string; description?: string }) {
  return (
    <Box sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
      <Typography variant="h6">{title}</Typography>
      {description && <Typography>{description}</Typography>}
    </Box>
  )
}
