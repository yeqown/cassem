import { useCallback, useEffect, useLayoutEffect, useRef, useState, type FormEvent } from 'react'
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline'
import FolderOpenIcon from '@mui/icons-material/FolderOpen'
import LayersIcon from '@mui/icons-material/Layers'
import {
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
import type { EnvsResponse } from '../../domain/types'
import { ApiError, apiRequest, buildQuery } from '../../lib/api'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback
}

async function requestEnvs(appId: string) {
  return apiRequest<EnvsResponse>(`/api/apps/${encodeURIComponent(appId)}/envs${buildQuery({ limit: 100 })}`)
}

export function EnvsPage() {
  const { appId = '' } = useParams()
  const [envs, setEnvs] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState('')
  const [envName, setEnvName] = useState('')
  const requestSeq = useRef(0)
  const mountedRef = useRef(false)
  const appIdRef = useRef(appId)

  useLayoutEffect(() => {
    appIdRef.current = appId
  }, [appId])

  const canApplyMutationResult = useCallback(
    (startedAppId: string) => mountedRef.current && appIdRef.current === startedAppId,
    [],
  )

  const loadEnvs = useCallback(async () => {
    const requestId = ++requestSeq.current

    if (!appId) {
      if (mountedRef.current && requestId === requestSeq.current) {
        setEnvs([])
        setError('missing app id')
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
      setError(getErrorMessage(err, 'failed to load environments'))
    } finally {
      if (mountedRef.current && requestId === requestSeq.current) setLoading(false)
    }
  }, [appId])

  useEffect(() => {
    mountedRef.current = true

    queueMicrotask(() => {
      void loadEnvs()
    })

    return () => {
      mountedRef.current = false
    }
  }, [loadEnvs])

  function closeCreateDialog() {
    if (submitting) return
    setCreateOpen(false)
    setEnvName('')
  }

  function closeDeleteDialog() {
    if (submitting) return
    setDeleteTarget('')
  }

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const startedAppId = appId
    const nextEnvName = envName.trim()
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
        setError(getErrorMessage(err, 'failed to create environment'))
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
        setError(getErrorMessage(err, 'failed to delete environment'))
      }
    } finally {
      if (mountedRef.current) setSubmitting(false)
    }
  }

  return (
    <Stack spacing={3}>
      <AppBreadcrumbs items={[{ label: 'Apps', to: '/apps' }, { label: appId || 'unknown', to: `/apps/${encodeURIComponent(appId)}/envs` }, { label: 'Environments' }]} />

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} justifyContent="space-between" alignItems={{ md: 'center' }}>
        <Box>
          <Stack direction="row" spacing={1} alignItems="center">
            <FolderOpenIcon color="primary" />
            <Typography variant="h4" component="h1">
              Environments
            </Typography>
          </Stack>
        </Box>
        <Button variant="contained" startIcon={<AddCircleOutlineIcon />} onClick={() => setCreateOpen(true)} disabled={submitting || !appId}>
          Add environment
        </Button>
      </Stack>

      {error && <ErrorState message={error} />}

      <DangerConfirmDialog
        open={Boolean(deleteTarget)}
        title="Delete environment"
        description={<>This will delete environment <strong>{deleteTarget}</strong> from app <strong>{appId}</strong>.</>}
        confirmLabel="Delete"
        loading={submitting}
        onClose={closeDeleteDialog}
        onConfirm={() => void handleDelete()}
      />

      <Dialog open={createOpen} onClose={closeCreateDialog} fullWidth maxWidth="sm">
        <Box component="form" onSubmit={(event) => void handleCreate(event)}>
          <DialogTitle>Add environment</DialogTitle>
          <DialogContent>
            <Stack spacing={2} sx={{ mt: 1 }}>
              <TextField label="Environment" value={envName} onChange={(event) => setEnvName(event.target.value)} required fullWidth disabled={submitting || !appId} helperText="Short environment name, for example dev, staging, or prod." />
            </Stack>
          </DialogContent>
          <DialogActions>
            <Button onClick={closeCreateDialog} disabled={submitting}>Cancel</Button>
            <Button type="submit" variant="contained" disabled={submitting || !appId || !envName.trim()}>Create</Button>
          </DialogActions>
        </Box>
      </Dialog>

      {loading ? (
        <LoadingState label="Loading environments" />
      ) : error ? null : envs.length === 0 ? (
        <EmptyState title="No environments found" description="Create an environment for this app to manage its elements." />
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Environment</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {envs.map((env) => (
                <TableRow key={env} hover>
                  <TableCell>{env}</TableCell>
                  <TableCell align="right">
                    <Stack direction="row" spacing={1} justifyContent="flex-end">
                      <Button component={RouterLink} to={`/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements`} size="small" startIcon={<LayersIcon />}>
                        Elements
                      </Button>
                      <Button color="error" size="small" startIcon={<DeleteOutlineIcon />} onClick={() => setDeleteTarget(env)} disabled={submitting}>
                        Delete
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
