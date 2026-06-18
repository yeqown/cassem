import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline'
import AppsIcon from '@mui/icons-material/Apps'
import DnsIcon from '@mui/icons-material/Dns'
import SearchIcon from '@mui/icons-material/Search'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  MenuItem,
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

const pageSizeOptions = [15, 30, 50, 100]

function formatCreatedAt(createdAt?: number) {
  if (!createdAt) return '-'
  return new Date(createdAt * 1000).toLocaleString()
}

async function requestApps(limit: number, seek: string, query: string) {
  return apiRequest<AppsResponse>(`/api/apps${buildQuery({ limit, query: query || undefined, seek: seek || undefined })}`)
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
  const [queryInput, setQueryInput] = useState('')
  const [query, setQuery] = useState('')
  const [pageSize, setPageSize] = useState(15)
  const [seek, setSeek] = useState('')
  const [seekStack, setSeekStack] = useState<string[]>([])
  const [pageIndex, setPageIndex] = useState(1)
  const [hasMore, setHasMore] = useState(false)
  const [nextSeek, setNextSeek] = useState('')
  const requestSeq = useRef(0)
  const mountedRef = useRef(false)
  const lastLoadKeyRef = useRef('')

  const loadApps = useCallback(async () => {
    const requestId = ++requestSeq.current

    try {
      const data = await requestApps(pageSize, seek, query)
      if (!mountedRef.current || requestId !== requestSeq.current) return
      setApps(data.apps || [])
      setHasMore(Boolean(data.hasMore))
      setNextSeek(data.nextSeek || '')
      setError('')
    } catch (err) {
      if (!mountedRef.current || requestId !== requestSeq.current) return
      setApps([])
      setHasMore(false)
      setNextSeek('')
      setError(getErrorMessage(err, 'failed to load apps'))
    } finally {
      if (mountedRef.current && requestId === requestSeq.current) setLoading(false)
    }
  }, [pageSize, query, seek])

  useEffect(() => {
    mountedRef.current = true

    const loadKey = JSON.stringify({ pageSize, query, seek })
    if (lastLoadKeyRef.current !== loadKey) {
      lastLoadKeyRef.current = loadKey
      void loadApps()
    }

    return () => {
      mountedRef.current = false
    }
  }, [loadApps, pageSize, query, seek])

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

  function resetPaging() {
    setSeek('')
    setSeekStack([])
    setPageIndex(1)
  }

  function handleSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setQuery(queryInput.trim())
    resetPaging()
  }

  function handleClearSearch() {
    setQueryInput('')
    setQuery('')
    resetPaging()
  }

  function handleNextPage() {
    if (!hasMore || !nextSeek) return
    setSeekStack((items) => [...items, seek])
    setSeek(nextSeek)
    setPageIndex((value) => value + 1)
  }

  function handlePreviousPage() {
    setSeekStack((items) => {
      if (items.length === 0) return items
      const nextStack = items.slice(0, -1)
      setSeek(items[items.length - 1])
      setPageIndex((value) => Math.max(1, value - 1))
      return nextStack
    })
  }

  function handlePageSizeChange(value: string) {
    setPageSize(Number(value))
    resetPaging()
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
      resetPaging()
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
      resetPaging()
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
          <Stack direction="row" spacing={1} alignItems="center">
            <AppsIcon data-testid="apps-title-icon" color="primary" />
            <Typography variant="h4" component="h1">
              Apps
            </Typography>
          </Stack>
          <Typography color="text.secondary">Manage application namespaces and navigate to their environments.</Typography>
        </Box>
        <Button variant="contained" startIcon={<AddCircleOutlineIcon />} onClick={() => setCreateOpen(true)} disabled={submitting}>
          Add app
        </Button>
      </Stack>

      {error && <ErrorState message={error} />}

      <Stack component="form" onSubmit={handleSearch} direction={{ xs: 'column', md: 'row' }} spacing={2} alignItems={{ md: 'center' }}>
        <TextField
          label="Search apps"
          value={queryInput}
          onChange={(event) => setQueryInput(event.target.value)}
          size="small"
          fullWidth
          disabled={loading}
        />
        <Button type="submit" variant="contained" startIcon={<SearchIcon />} disabled={loading} sx={{ minWidth: 128, height: 40, px: 3 }}>
          Search
        </Button>
        <Button variant="text" onClick={handleClearSearch} disabled={loading || (!query && !queryInput)}>
          Clear
        </Button>
        <TextField
          select
          label="Rows per page"
          value={String(pageSize)}
          onChange={(event) => handlePageSizeChange(event.target.value)}
          size="small"
          sx={{ minWidth: 140 }}
          disabled={loading}
        >
          {pageSizeOptions.map((option) => (
            <MenuItem key={option} value={option}>{option}</MenuItem>
          ))}
        </TextField>
      </Stack>

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
      ) : apps.length === 0 ? (
        error ? null : <EmptyState title="No apps found" description="Create an app to start organizing environments and elements." />
      ) : (
        <Paper>
          <TableContainer>
            <Table>
            <TableHead>
              <TableRow>
                <TableCell>App ID</TableCell>
                <TableCell>Description</TableCell>
                <TableCell>Created At</TableCell>
                <TableCell>Creator</TableCell>
                <TableCell>Owner</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {apps.map((app) => (
                <TableRow key={app.id} hover>
                  <TableCell>{app.id}</TableCell>
                  <TableCell>{app.description || '-'}</TableCell>
                  <TableCell>{formatCreatedAt(app.createdAt)}</TableCell>
                  <TableCell>{app.creator || '-'}</TableCell>
                  <TableCell>{app.owner || '-'}</TableCell>
                  <TableCell align="right">
                    <Stack direction="row" spacing={1} justifyContent="flex-end">
                      <Button component={RouterLink} to={`/apps/${encodeURIComponent(app.id)}/envs`} size="small" startIcon={<DnsIcon />}>
                        Envs
                      </Button>
                      <Button color="error" size="small" startIcon={<DeleteOutlineIcon />} onClick={() => setDeleteTarget(app)} disabled={submitting}>
                        Delete
                      </Button>
                    </Stack>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
            </Table>
          </TableContainer>
          <Stack direction="row" spacing={2} justifyContent="flex-end" alignItems="center" sx={{ p: 2 }}>
            <Typography color="text.secondary">Page {pageIndex}</Typography>
            <Button onClick={handlePreviousPage} disabled={loading || seekStack.length === 0}>Previous</Button>
            <Button onClick={handleNextPage} disabled={loading || !hasMore || !nextSeek}>Next</Button>
          </Stack>
        </Paper>
      )}
    </Stack>
  )
}
