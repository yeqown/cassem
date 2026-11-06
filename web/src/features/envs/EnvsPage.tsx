import { useCallback, useEffect, useLayoutEffect, useRef, useState, type FormEvent } from 'react'
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline'
import ContentCopyIcon from '@mui/icons-material/ContentCopy'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline'
import FolderOpenIcon from '@mui/icons-material/FolderOpen'
import LayersIcon from '@mui/icons-material/Layers'
import {
  Autocomplete,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Paper,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material'
import { Link as RouterLink, useParams } from 'react-router-dom'
import { AppBreadcrumbs } from '../../components/AppBreadcrumbs'
import { DangerConfirmDialog } from '../../components/DangerConfirmDialog'
import { EmptyState, ErrorState, LoadingState } from '../../components/StateView'
import { useErrorState } from '../../components/useErrorState'
import type { EnvsResponse } from '../../domain/types'
import { useI18n } from '../../i18n'
import { ApiError, apiRequest, buildQuery } from '../../lib/api'
import { CopyEnvDialog } from './CopyEnvDialog'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback
}

const envNameOptions = ['dev', 'test', 'stage', 'prod']

async function requestEnvs(appId: string) {
  return apiRequest<EnvsResponse>(`/api/apps/${encodeURIComponent(appId)}/envs${buildQuery({ limit: 100 })}`)
}

export function EnvsPage() {
  const { t } = useI18n()
  const { appId = '' } = useParams()
  const [envs, setEnvs] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useErrorState()
  const [createOpen, setCreateOpen] = useState(false)
  const [copyOpen, setCopyOpen] = useState(false)
  const [copyBusy, setCopyBusy] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState('')
  const [envName, setEnvName] = useState('')
  const requestSeq = useRef(0)
  const mountedRef = useRef(false)
  const lastLoadKeyRef = useRef('')
  const appIdRef = useRef(appId)

  useLayoutEffect(() => {
    appIdRef.current = appId
  }, [appId, setError])

  const canApplyMutationResult = useCallback(
    (startedAppId: string) => mountedRef.current && appIdRef.current === startedAppId,
    [],
  )

  const loadEnvs = useCallback(async () => {
    const requestId = ++requestSeq.current

    if (!appId) {
      if (mountedRef.current && requestId === requestSeq.current) {
        setEnvs([])
        setError(t('management.envs.missingApp'))
        setLoading(false)
      }
      return
    }

    try {
      const data = await requestEnvs(appId)
      if (!mountedRef.current || requestId !== requestSeq.current) return
      setEnvs(data.envs || [])
      setError('')
    } catch (err) {
      if (!mountedRef.current || requestId !== requestSeq.current) return
      setEnvs([])
      setError(getErrorMessage(err, t('management.envs.loadFailed')))
    } finally {
      if (mountedRef.current && requestId === requestSeq.current) setLoading(false)
    }
  }, [appId, setError, t])

  useEffect(() => {
    mountedRef.current = true

    const loadKey = JSON.stringify({ appId })
    if (lastLoadKeyRef.current !== loadKey) {
      lastLoadKeyRef.current = loadKey
      queueMicrotask(() => {
        void loadEnvs()
      })
    }

    return () => {
      mountedRef.current = false
    }
  }, [appId, loadEnvs])

  function closeCreateDialog() {
    if (submitting || copyBusy) return
    setCreateOpen(false)
    setEnvName('')
  }

  function closeDeleteDialog() {
    if (submitting || copyBusy) return
    setDeleteTarget('')
  }

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const startedAppId = appId
    const nextEnvName = envName.trim().toLowerCase()
    if (!startedAppId || !nextEnvName) return

    setSubmitting(true)
    setError('')

    try {
      await apiRequest<void>(`/api/apps/${encodeURIComponent(startedAppId)}/envs/${encodeURIComponent(nextEnvName)}`, {
        method: 'POST',
      })

      if (!canApplyMutationResult(startedAppId)) return

      setCreateOpen(false)
      setEnvName('')
      await loadEnvs()
    } catch (err) {
      if (canApplyMutationResult(startedAppId)) {
        setError(getErrorMessage(err, t('management.envs.createFailed')))
      }
    } finally {
      if (mountedRef.current) setSubmitting(false)
    }
  }

  async function handleDelete() {
    const startedAppId = appId
    const targetEnvName = deleteTarget
    if (!startedAppId || !targetEnvName) return

    setSubmitting(true)
    setError('')

    try {
      await apiRequest<void>(`/api/apps/${encodeURIComponent(startedAppId)}/envs/${encodeURIComponent(targetEnvName)}`, {
        method: 'DELETE',
      })

      if (!canApplyMutationResult(startedAppId)) return

      setDeleteTarget('')
      await loadEnvs()
    } catch (err) {
      if (canApplyMutationResult(startedAppId)) {
        setError(getErrorMessage(err, t('management.envs.deleteFailed')))
      }
    } finally {
      if (mountedRef.current) setSubmitting(false)
    }
  }

  return (
    <Stack spacing={3}>
      <AppBreadcrumbs items={[{ label: t('management.apps.title'), to: '/apps' }, { label: appId || t('management.common.unknown'), to: `/apps/${encodeURIComponent(appId)}/envs` }, { label: t('management.envs.title') }]} />

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} justifyContent="space-between" alignItems={{ md: 'center' }}>
        <Box>
          <Stack direction="row" spacing={1} alignItems="center">
            <FolderOpenIcon color="primary" />
            <Typography variant="h4" component="h1">
              {t('management.envs.title')}
            </Typography>
          </Stack>
          <Typography color="text.secondary">{t('management.envs.description')}</Typography>
        </Box>
        <Stack direction="row" spacing={1}>
          <Button
            variant="outlined"
            startIcon={<ContentCopyIcon />}
            onClick={() => setCopyOpen(true)}
            disabled={submitting || copyBusy || !appId || envs.length === 0}
          >
            {t('management.envs.copy')}
          </Button>
          <Button variant="contained" startIcon={<AddCircleOutlineIcon />} onClick={() => setCreateOpen(true)} disabled={submitting || copyBusy || !appId}>
            {t('management.envs.add')}
          </Button>
        </Stack>
      </Stack>

      {error.message && <ErrorState message={error.message} eventKey={error.eventKey} />}

      <DangerConfirmDialog
        open={Boolean(deleteTarget)}
        title={t('management.envs.delete')}
        description={t('management.envs.deleteDescription', { env: deleteTarget, app: appId })}
        confirmLabel={t('management.common.delete')}
        loading={submitting || copyBusy}
        onClose={closeDeleteDialog}
        onConfirm={() => void handleDelete()}
      />

      <CopyEnvDialog
        open={copyOpen}
        appId={appId}
        envs={envs}
        onClose={() => setCopyOpen(false)}
        onBusyChange={setCopyBusy}
        onFinished={loadEnvs}
      />

      <Dialog open={createOpen} onClose={closeCreateDialog} fullWidth maxWidth="sm">
        <Box component="form" onSubmit={(event) => void handleCreate(event)}>
          <DialogTitle>{t('management.envs.add')}</DialogTitle>
          <DialogContent>
            <Stack spacing={2} sx={{ mt: 1 }}>
              <Autocomplete
                freeSolo
                options={envNameOptions}
                value={envName}
                inputValue={envName}
                onChange={(_, value) => setEnvName(String(value || '').toLowerCase())}
                onInputChange={(_, value) => setEnvName(value.toLowerCase())}
                disabled={submitting || !appId}
                renderInput={(params) => (
                  <TextField
                    {...params}
                    label={t('management.common.environment')}
                    required
                    fullWidth
                    helperText={t('management.envs.nameHelp')}
                  />
                )}
              />
            </Stack>
          </DialogContent>
          <DialogActions>
            <Button onClick={closeCreateDialog} disabled={submitting}>{t('management.common.cancel')}</Button>
            <Button type="submit" variant="contained" disabled={submitting || !appId || !envName.trim()}>{t('management.common.create')}</Button>
          </DialogActions>
        </Box>
      </Dialog>

      {loading ? (
        <LoadingState label={t('management.envs.loading')} />
      ) : envs.length === 0 ? (
        error.message ? null : <EmptyState title={t('management.envs.empty')} description={t('management.envs.emptyDescription')} />
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>{t('management.common.environment')}</TableCell>
                <TableCell align="right">{t('management.common.actions')}</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {envs.map((env) => (
                <TableRow key={env} hover>
                  <TableCell>{env}</TableCell>
                  <TableCell align="right">
                    <Stack direction="row" spacing={1} justifyContent="flex-end">
                      <Button component={RouterLink} to={`/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements`} size="small" startIcon={<LayersIcon />}>
                        {t('management.envs.elements')}
                      </Button>
                      <Button color="error" size="small" startIcon={<DeleteOutlineIcon />} onClick={() => setDeleteTarget(env)} disabled={submitting || copyBusy}>
                        {t('management.common.delete')}
                      </Button>
                    </Stack>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </Stack>
  )
}
