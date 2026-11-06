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
import { useErrorState } from '../../components/useErrorState'
import type { AppMetadata, AppsResponse } from '../../domain/types'
import { useI18n } from '../../i18n'
import { ApiError, apiRequest, buildQuery, jsonBody } from '../../lib/api'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback
}

const pageSizeOptions = [15, 30, 50, 100]

async function requestApps(limit: number, seek: string, query: string) {
  return apiRequest<AppsResponse>(`/api/apps${buildQuery({ limit, query: query || undefined, seek: seek || undefined })}`)
}

export function AppsPage() {
  const { t, formatDateTime, formatNumber } = useI18n()
  const [apps, setApps] = useState<AppMetadata[]>([])
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useErrorState()
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
      setError(getErrorMessage(err, t('management.apps.loadFailed')))
    } finally {
      if (mountedRef.current && requestId === requestSeq.current) setLoading(false)
    }
  }, [pageSize, query, seek, setError, t])

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
      setError(getErrorMessage(err, t('management.apps.createFailed')))
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
      setError(getErrorMessage(err, t('management.apps.deleteFailed')))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Stack spacing={3}>
      <AppBreadcrumbs items={[{ label: t('management.apps.title') }]} />

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} justifyContent="space-between" alignItems={{ md: 'center' }}>
        <Box>
          <Stack direction="row" spacing={1} alignItems="center">
            <AppsIcon data-testid="apps-title-icon" color="primary" />
            <Typography variant="h4" component="h1">
              {t('management.apps.title')}
            </Typography>
          </Stack>
          <Typography color="text.secondary">{t('management.apps.description')}</Typography>
        </Box>
        <Button variant="contained" startIcon={<AddCircleOutlineIcon />} onClick={() => setCreateOpen(true)} disabled={submitting}>
          {t('management.apps.add')}
        </Button>
      </Stack>

      {error.message && <ErrorState message={error.message} eventKey={error.eventKey} />}

      <Stack component="form" onSubmit={handleSearch} direction={{ xs: 'column', md: 'row' }} spacing={2} alignItems={{ md: 'center' }}>
        <TextField
          label={t('management.apps.search')}
          value={queryInput}
          onChange={(event) => setQueryInput(event.target.value)}
          size="small"
          fullWidth
          disabled={loading}
        />
        <Button type="submit" variant="contained" startIcon={<SearchIcon />} disabled={loading} sx={{ minWidth: 128, height: 40, px: 3 }}>
          {t('management.common.search')}
        </Button>
        <Button variant="text" onClick={handleClearSearch} disabled={loading || (!query && !queryInput)}>
          {t('management.common.clear')}
        </Button>
        <TextField
          select
          label={t('management.common.rowsPerPage')}
          value={String(pageSize)}
          onChange={(event) => handlePageSizeChange(event.target.value)}
          size="small"
          sx={{ minWidth: 140 }}
          disabled={loading}
        >
          {pageSizeOptions.map((option) => (
            <MenuItem key={option} value={option}>{formatNumber(option)}</MenuItem>
          ))}
        </TextField>
      </Stack>

      <DangerConfirmDialog
        open={Boolean(deleteTarget)}
        title={t('management.apps.delete')}
        description={t('management.apps.deleteDescription', { app: deleteTarget?.id || '' })}
        confirmLabel={t('management.common.delete')}
        loading={submitting}
        onClose={closeDeleteDialog}
        onConfirm={() => void handleDelete()}
      />

      <Dialog open={createOpen} onClose={closeCreateDialog} fullWidth maxWidth="sm">
        <Box component="form" onSubmit={(event) => void handleCreate(event)}>
          <DialogTitle>{t('management.apps.add')}</DialogTitle>
          <DialogContent>
            <Stack spacing={2} sx={{ mt: 1 }}>
              <TextField
                label={t('management.common.appId')}
                value={appId}
                onChange={(event) => setAppId(event.target.value)}
                required
                fullWidth
                disabled={submitting}
              />
              <TextField
                label={t('management.common.description')}
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
              {t('management.common.cancel')}
            </Button>
            <Button type="submit" variant="contained" disabled={submitting || !appId.trim() || !description.trim()}>
              {t('management.common.create')}
            </Button>
          </DialogActions>
        </Box>
      </Dialog>

      {loading ? (
        <LoadingState label={t('management.apps.loading')} />
      ) : apps.length === 0 ? (
        error.message ? null : <EmptyState title={t('management.apps.empty')} description={t('management.apps.emptyDescription')} />
      ) : (
        <Paper>
          <TableContainer>
            <Table>
            <TableHead>
              <TableRow>
                <TableCell>{t('management.common.appId')}</TableCell>
                <TableCell>{t('management.common.description')}</TableCell>
                <TableCell>{t('management.common.createdAt')}</TableCell>
                <TableCell>{t('management.common.creator')}</TableCell>
                <TableCell>{t('management.common.owner')}</TableCell>
                <TableCell align="right">{t('management.common.actions')}</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {apps.map((app) => (
                <TableRow key={app.id} hover>
                  <TableCell>{app.id}</TableCell>
                  <TableCell>{app.description || '-'}</TableCell>
                  <TableCell>{app.createdAt ? formatDateTime(app.createdAt * 1000) : '-'}</TableCell>
                  <TableCell>{app.creator || '-'}</TableCell>
                  <TableCell>{app.owner || '-'}</TableCell>
                  <TableCell align="right">
                    <Stack direction="row" spacing={1} justifyContent="flex-end">
                      <Button component={RouterLink} to={`/apps/${encodeURIComponent(app.id)}/envs`} size="small" startIcon={<DnsIcon />}>
                        {t('management.apps.envs')}
                      </Button>
                      <Button color="error" size="small" startIcon={<DeleteOutlineIcon />} onClick={() => setDeleteTarget(app)} disabled={submitting}>
                        {t('management.common.delete')}
                      </Button>
                    </Stack>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
            </Table>
          </TableContainer>
          <Stack direction="row" spacing={2} justifyContent="flex-end" alignItems="center" sx={{ p: 2 }}>
            <Typography color="text.secondary">{t('management.common.page', { page: formatNumber(pageIndex) })}</Typography>
            <Button onClick={handlePreviousPage} disabled={loading || seekStack.length === 0}>{t('management.common.previous')}</Button>
            <Button onClick={handleNextPage} disabled={loading || !hasMore || !nextSeek}>{t('management.common.next')}</Button>
          </Stack>
        </Paper>
      )}
    </Stack>
  )
}
