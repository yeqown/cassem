import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline'
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
import { Link as RouterLink } from 'react-router-dom'
import { AppBreadcrumbs } from '../../components/AppBreadcrumbs'
import { DangerConfirmDialog } from '../../components/DangerConfirmDialog'
import { EmptyState, ErrorState, LoadingState } from '../../components/StateView'
import type { AppMetadata, AppsResponse } from '../../domain/types'
import { ApiError, apiRequest, buildQuery, jsonBody } from '../../lib/api'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback
}

async function requestApps() {
  return apiRequest<AppsResponse>(`/api/apps${buildQuery({ limit: 100 })}`)
}

export function AppsPage() {
  const [apps, setApps] = useState<AppMetadata[]>([])
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<AppMetadata | null>(null)
  const [appId, setAppId] = useState('')
  const [description, setDescription] = useState('')
  const requestSeq = useRef(0)
  const mountedRef = useRef(false)

  const loadApps = useCallback(async () => {
    const requestId = ++requestSeq.current

    try {
      const data = await requestApps()
      if (!mountedRef.current || requestId !== requestSeq.current) return
      setApps(data.apps || [])
      setError('')
    } catch (err) {
      if (!mountedRef.current || requestId !== requestSeq.current) return
      setApps([])
      setError(getErrorMessage(err, 'failed to load apps'))
    } finally {
      if (mountedRef.current && requestId === requestSeq.current) setLoading(false)
    }
  }, [])

  useEffect(() => {
    mountedRef.current = true

    queueMicrotask(() => {
      void loadApps()
    })

    return () => {
      mountedRef.current = false
    }
  }, [loadApps])

  function closeCreateDialog() {
    if (submitting) return
    setCreateOpen(false)
    setAppId('')
    setDescription('')
  }

  function closeDeleteDialog() {
    if (submitting) return
    setDeleteTarget(null)
  }

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const nextAppId = appId.trim()
    const nextDescription = description.trim()
    if (!nextAppId || !nextDescription) return

    setSubmitting(true)
    setError('')

    try {
      await apiRequest<void>(
        `/api/apps/${encodeURIComponent(nextAppId)}`,
        jsonBody({ name: nextAppId, description: nextDescription }),
      )
      setCreateOpen(false)
      setAppId('')
      setDescription('')
      await loadApps()
    } catch (err) {
      setError(getErrorMessage(err, 'failed to create app'))
    } finally {
      setSubmitting(false)
    }
  }

  async function handleDelete() {
    const targetAppId = deleteTarget?.id
    if (!targetAppId) return

    setSubmitting(true)
    setError('')

    try {
      await apiRequest<void>(`/api/apps/${encodeURIComponent(targetAppId)}`, { method: 'DELETE' })
      setDeleteTarget(null)
      await loadApps()
    } catch (err) {
      setError(getErrorMessage(err, 'failed to delete app'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Stack spacing={3}>
      <AppBreadcrumbs items={[{ label: 'Apps' }]} />

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} justifyContent="space-between" alignItems={{ md: 'center' }}>
        <Box>
          <Typography variant="h4" component="h1">
            Apps
          </Typography>
          <Typography color="text.secondary">Manage application namespaces and navigate to their environments.</Typography>
        </Box>
        <Button variant="contained" startIcon={<AddCircleOutlineIcon />} onClick={() => setCreateOpen(true)} disabled={submitting}>
          Add app
        </Button>
      </Stack>

      {error && <ErrorState message={error} />}

      <DangerConfirmDialog
        open={Boolean(deleteTarget)}
        title="Delete app"
        description={<>This will delete app <strong>{deleteTarget?.id}</strong>.</>}
        confirmLabel="Delete"
        loading={submitting}
        onClose={closeDeleteDialog}
        onConfirm={() => void handleDelete()}
      />

      <Dialog open={createOpen} onClose={closeCreateDialog} fullWidth maxWidth="sm">
        <Box component="form" onSubmit={(event) => void handleCreate(event)}>
          <DialogTitle>Add app</DialogTitle>
          <DialogContent>
            <Stack spacing={2} sx={{ mt: 1 }}>
              <TextField
                label="App ID"
                value={appId}
                onChange={(event) => setAppId(event.target.value)}
                required
                fullWidth
                disabled={submitting}
              />
              <TextField
                label="Description"
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                required
                fullWidth
                disabled={submitting}
              />
            </Stack>
          </DialogContent>
          <DialogActions>
            <Button onClick={closeCreateDialog} disabled={submitting}>
              Cancel
            </Button>
            <Button type="submit" variant="contained" disabled={submitting || !appId.trim() || !description.trim()}>
              Create
            </Button>
          </DialogActions>
        </Box>
      </Dialog>

      {loading ? (
        <LoadingState label="Loading apps" />
      ) : error ? null : apps.length === 0 ? (
        <EmptyState title="No apps found" description="Create an app to start organizing environments and elements." />
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>App</TableCell>
                <TableCell>Description</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {apps.map((app) => (
                <TableRow key={app.id} hover>
                  <TableCell>{app.id}</TableCell>
                  <TableCell>{app.description || '-'}</TableCell>
                  <TableCell align="right">
                    <Stack direction="row" spacing={1} justifyContent="flex-end">
                      <Button component={RouterLink} to={`/apps/${encodeURIComponent(app.id)}/envs`} size="small">
                        Envs
                      </Button>
                      <Button color="error" size="small" onClick={() => setDeleteTarget(app)} disabled={submitting}>
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
