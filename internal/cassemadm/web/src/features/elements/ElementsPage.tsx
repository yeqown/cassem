import { useCallback, useEffect, useLayoutEffect, useRef, useState, type FormEvent } from 'react'
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline'
import FilterAltIcon from '@mui/icons-material/FilterAlt'
import Inventory2Icon from '@mui/icons-material/Inventory2'
import OpenInNewIcon from '@mui/icons-material/OpenInNew'
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
import { Link as RouterLink, useParams } from 'react-router-dom'
import { AppBreadcrumbs } from '../../components/AppBreadcrumbs'
import { DangerConfirmDialog } from '../../components/DangerConfirmDialog'
import { EmptyState, ErrorState, LoadingState } from '../../components/StateView'
import { contentTypes, formatVersionLabel, type Element, type ElementsResponse } from '../../domain/types'
import { ApiError, apiRequest, buildQuery, jsonBody } from '../../lib/api'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback
}

function getContentTypeLabel(contentType?: number | string) {
  const value = Number(contentType)
  return contentTypes.find((item) => item.value === value)?.label || String(contentType || '-')
}

async function requestElements(appId: string, env: string, key: string) {
  return apiRequest<ElementsResponse>(
    `/api/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements${buildQuery({ limit: 100, key: key || undefined })}`,
  )
}

export function ElementsPage() {
  const { appId = '', env = '' } = useParams()
  const [elements, setElements] = useState<Element[]>([])
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [filterInput, setFilterInput] = useState('')
  const [filterKey, setFilterKey] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState('')
  const [createKey, setCreateKey] = useState('')
  const [createRaw, setCreateRaw] = useState('')
  const [createContentType, setCreateContentType] = useState<number>(contentTypes[0].value)
  const requestSeq = useRef(0)
  const mountedRef = useRef(false)
  const appIdRef = useRef(appId)
  const envRef = useRef(env)
  const filterKeyRef = useRef(filterKey)

  useLayoutEffect(() => {
    appIdRef.current = appId
    envRef.current = env
    filterKeyRef.current = filterKey
  }, [appId, env, filterKey])

  const canApplyMutationResult = useCallback(
    (startedAppId: string, startedEnv: string) => mountedRef.current && appIdRef.current === startedAppId && envRef.current === startedEnv,
    [],
  )

  const loadElements = useCallback(
    async (targetFilter = filterKeyRef.current) => {
      const requestId = ++requestSeq.current

      if (!appId || !env) {
        if (mountedRef.current && requestId === requestSeq.current) {
          setElements([])
          setError('missing app id or environment')
          setLoading(false)
        }
        return
      }

      try {
        const data = await requestElements(appId, env, targetFilter)
        if (!mountedRef.current || requestId !== requestSeq.current) return
        setElements(data.elements || [])
        setError('')
      } catch (err) {
        if (!mountedRef.current || requestId !== requestSeq.current) return
        setElements([])
        setError(getErrorMessage(err, 'failed to load elements'))
      } finally {
        if (mountedRef.current && requestId === requestSeq.current) setLoading(false)
      }
    },
    [appId, env],
  )

  useEffect(() => {
    mountedRef.current = true

    queueMicrotask(() => {
      void loadElements()
    })

    return () => {
      mountedRef.current = false
    }
  }, [loadElements])

  async function handleFilterSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const nextFilter = filterInput.trim()
    filterKeyRef.current = nextFilter
    setFilterKey(nextFilter)
    setLoading(true)
    void loadElements(nextFilter)
  }

  function closeCreateDialog() {
    if (submitting) return
    setCreateOpen(false)
    setCreateKey('')
    setCreateRaw('')
    setCreateContentType(contentTypes[0].value)
  }

  function closeDeleteDialog() {
    if (submitting) return
    setDeleteTarget('')
  }

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const startedAppId = appId
    const startedEnv = env
    const nextKey = createKey.trim()
    if (!startedAppId || !startedEnv || !nextKey || !createRaw.trim()) return

    setSubmitting(true)
    setError('')

    try {
      await apiRequest<void>(
        `/api/apps/${encodeURIComponent(startedAppId)}/envs/${encodeURIComponent(startedEnv)}/elements/${encodeURIComponent(nextKey)}`,
        jsonBody({ raw: createRaw, contentType: createContentType }),
      )

      if (!canApplyMutationResult(startedAppId, startedEnv)) return

      setCreateOpen(false)
      setCreateKey('')
      setCreateRaw('')
      setCreateContentType(contentTypes[0].value)
      setLoading(true)
      await loadElements(filterKey)
    } catch (err) {
      if (canApplyMutationResult(startedAppId, startedEnv)) {
        setError(getErrorMessage(err, 'failed to create element'))
      }
    } finally {
      if (mountedRef.current) setSubmitting(false)
    }
  }

  async function handleDelete() {
    const startedAppId = appId
    const startedEnv = env
    const targetKey = deleteTarget
    if (!startedAppId || !startedEnv || !targetKey) return

    setSubmitting(true)
    setError('')

    try {
      await apiRequest<void>(
        `/api/apps/${encodeURIComponent(startedAppId)}/envs/${encodeURIComponent(startedEnv)}/elements/${encodeURIComponent(targetKey)}`,
        { method: 'DELETE' },
      )

      if (!canApplyMutationResult(startedAppId, startedEnv)) return

      setDeleteTarget('')
      setLoading(true)
      await loadElements(filterKey)
    } catch (err) {
      if (canApplyMutationResult(startedAppId, startedEnv)) {
        setError(getErrorMessage(err, 'failed to delete element'))
      }
    } finally {
      if (mountedRef.current) setSubmitting(false)
    }
  }

  return (
    <Stack spacing={3}>
      <AppBreadcrumbs items={[
        { label: 'Apps', to: '/apps' },
        { label: appId || 'unknown', to: `/apps/${encodeURIComponent(appId)}/envs` },
        { label: env || 'unknown', to: `/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements` },
        { label: 'Elements' },
      ]} />

      <Stack direction={{ xs: 'column', md: 'row' }} justifyContent="space-between" spacing={2} alignItems={{ md: 'center' }}>
        <Box>
          <Stack direction="row" spacing={1} alignItems="center">
            <Inventory2Icon color="primary" />
            <Typography variant="h4" component="h1">
              Elements
            </Typography>
          </Stack>
        </Box>
        <Button variant="contained" startIcon={<AddCircleOutlineIcon />} onClick={() => setCreateOpen(true)} disabled={!appId || !env || submitting}>
          Add element
        </Button>
      </Stack>

      {error && <ErrorState message={error} />}

      <DangerConfirmDialog
        open={Boolean(deleteTarget)}
        title="Delete element"
        description={<>This will delete element <strong>{deleteTarget}</strong> from <strong>{appId}/{env}</strong>.</>}
        confirmLabel="Delete"
        loading={submitting}
        onClose={closeDeleteDialog}
        onConfirm={() => void handleDelete()}
      />

      <Paper component="form" onSubmit={(event) => void handleFilterSubmit(event)} sx={{ p: 2.5 }}>
        <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} alignItems={{ md: 'center' }}>
          <TextField
            label="Exact key"
            value={filterInput}
            onChange={(event) => setFilterInput(event.target.value)}
            fullWidth
            helperText="Optional exact element key filter, for example db.url."
            disabled={loading || submitting || !appId || !env}
          />
          <Button type="submit" variant="outlined" startIcon={<FilterAltIcon />} disabled={loading || submitting || !appId || !env}>
            Filter
          </Button>
        </Stack>
      </Paper>

      <Dialog open={createOpen} onClose={closeCreateDialog} fullWidth maxWidth="md">
        <Box component="form" onSubmit={(event) => void handleCreate(event)}>
          <DialogTitle>Add element</DialogTitle>
          <DialogContent>
            <Stack spacing={2} sx={{ mt: 1 }}>
              <TextField label="Key" value={createKey} onChange={(event) => setCreateKey(event.target.value)} required fullWidth disabled={submitting} helperText="Configuration key name, for example db.url or redis.password." />
              <TextField
                select
                label="Content type"
                value={String(createContentType)}
                onChange={(event) => setCreateContentType(Number(event.target.value))}
                fullWidth
                disabled={submitting}
              >
                {contentTypes.map((contentType) => (
                  <MenuItem key={contentType.value} value={String(contentType.value)}>
                    {contentType.label}
                  </MenuItem>
                ))}
              </TextField>
              <TextField label="Raw" value={createRaw} onChange={(event) => setCreateRaw(event.target.value)} required fullWidth multiline minRows={10} disabled={submitting} helperText="Paste the full configuration payload for this element." />
            </Stack>
          </DialogContent>
          <DialogActions>
            <Button onClick={closeCreateDialog} disabled={submitting}>Cancel</Button>
            <Button type="submit" variant="contained" disabled={submitting || !createKey.trim() || !createRaw.trim()}>Create</Button>
          </DialogActions>
        </Box>
      </Dialog>

      {loading ? (
        <LoadingState label="Loading elements" />
      ) : error ? null : elements.length === 0 ? (
        <EmptyState title="No matching records" description={filterKey ? 'Try a different exact key filter or create a new element.' : 'Add an element to manage versioned configuration.'} />
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Key</TableCell>
                <TableCell>Latest</TableCell>
                <TableCell>Using</TableCell>
                <TableCell>Draft</TableCell>
                <TableCell>Type</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {elements.map((element) => {
                const metadata = element.metadata || { key: '' }
                return (
                  <TableRow key={metadata.key} hover>
                    <TableCell>{metadata.key || '-'}</TableCell>
                    <TableCell>{formatVersionLabel(metadata.latestVersion)}</TableCell>
                    <TableCell>{formatVersionLabel(metadata.usingVersion)}</TableCell>
                    <TableCell>{formatVersionLabel(metadata.unpublishedVersion)}</TableCell>
                    <TableCell>{getContentTypeLabel(metadata.contentType)}</TableCell>
                    <TableCell align="right">
                      <Stack direction="row" spacing={1} justifyContent="flex-end">
                        <Button component={RouterLink} to={`/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(metadata.key)}`} size="small" startIcon={<OpenInNewIcon />}>
                          Detail
                        </Button>
                        <Button color="error" size="small" startIcon={<DeleteOutlineIcon />} onClick={() => setDeleteTarget(metadata.key)} disabled={submitting}>
                          Delete
                        </Button>
                      </Stack>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </Stack>
  )
}
