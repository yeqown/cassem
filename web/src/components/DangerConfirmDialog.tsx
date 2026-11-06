import type { ReactNode } from 'react'
import { useI18n } from '../i18n'
import WarningAmberIcon from '@mui/icons-material/WarningAmber'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Stack,
} from '@mui/material'

type DangerConfirmDialogProps = {
  open: boolean
  title: string
  description: ReactNode
  confirmLabel: string
  loading?: boolean
  onClose: () => void
  onConfirm: () => void
}

export function DangerConfirmDialog({
  open,
  title,
  description,
  confirmLabel,
  loading = false,
  onClose,
  onConfirm,
}: DangerConfirmDialogProps) {
  const { t } = useI18n()
  return (
    <Dialog open={open} onClose={loading ? undefined : onClose} fullWidth maxWidth="sm">
      <DialogTitle>
        <Stack direction="row" spacing={1} alignItems="center">
          <WarningAmberIcon color="error" />
          <span>{title}</span>
        </Stack>
      </DialogTitle>
      <DialogContent>
        <DialogContentText component="div">{description}</DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={loading}>{t('common.cancel')}</Button>
        <Button variant="contained" color="error" onClick={onConfirm} disabled={loading}>
          {confirmLabel}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
