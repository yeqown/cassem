import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'
import CompareArrowsIcon from '@mui/icons-material/CompareArrows'
import Inventory2Icon from '@mui/icons-material/Inventory2'
import LaunchIcon from '@mui/icons-material/Launch'
import LockResetIcon from '@mui/icons-material/LockReset'
import PublishIcon from '@mui/icons-material/Publish'
import SaveIcon from '@mui/icons-material/Save'
import TimelineIcon from '@mui/icons-material/Timeline'
import {
  Box,
  Button,
  Divider,
  Paper,
  Stack,
  Tab,
  Tabs,
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
import { EmptyState, ErrorState, LoadingState } from '../../components/StateView'
import { contentTypes, type DiffResponse, type Element, type ElementOperation, type ElementOperationsResponse, type ElementsResponse } from '../../domain/types'
import { ApiError, apiRequest, buildQuery, jsonBody } from '../../lib/api'
import { decodeRaw } from '../../lib/raw'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback
}

function getContentTypeLabel(contentType?: number | string) {
  const value = Number(contentType)
  return contentTypes.find((item) => item.value === value)?.label || String(contentType || '-')
}

function formatTimestamp(timestamp?: number) {
  if (!timestamp) return '-'

  const milliseconds = timestamp > 1_000_000_000_000_000 ? Math.floor(timestamp / 1_000_000) : timestamp
  const date = new Date(milliseconds)
  return Number.isNaN(date.getTime()) ? '-' : date.toLocaleString()
}

async function requestElement(appId: string, env: string, key: string) {
  return apiRequest<Element>(`/api/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}`)
}

async function requestVersions(appId: string, env: string, key: string) {
  return apiRequest<ElementsResponse>(
    `/api/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}/versions${buildQuery({ limit: 100 })}`,
  )
}

async function requestOperations(appId: string, env: string, key: string) {
  return apiRequest<ElementOperationsResponse>(
    `/api/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}/operations${buildQuery({ limit: 100 })}`,
  )
}

export function ElementDetailPage() {
  const { appId = '', env = '', key = '' } = useParams()
  const [element, setElement] = useState<Element | null>(null)
  const [versions, setVersions] = useState<Element[]>([])
  const [operations, setOperations] = useState<ElementOperation[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [diffLoading, setDiffLoading] = useState(false)
  const [error, setError] = useState('')
  const [tab, setTab] = useState(0)
  const [raw, setRaw] = useState('')
  const [diffBase, setDiffBase] = useState('')
  const [diffCompare, setDiffCompare] = useState('')
  const [diffText, setDiffText] = useState('')
  const requestSeq = useRef(0)
  const diffRequestSeq = useRef(0)
  const mountedRef = useRef(false)
  const locationRef = useRef({ appId, env, key })

  useLayoutEffect(() => {
    locationRef.current = { appId, env, key }
  }, [appId, env, key])

  const canApplyMutationResult = useCallback(
    (startedAppId: string, startedEnv: string, startedKey: string) =>
      mountedRef.current &&
      locationRef.current.appId === startedAppId &&
      locationRef.current.env === startedEnv &&
      locationRef.current.key === startedKey,
    [],
  )

  const loadPage = useCallback(async () => {
    const requestId = ++requestSeq.current

    if (!appId || !env || !key) {
      if (mountedRef.current && requestId === requestSeq.current) {
        setElement(null)
        setVersions([])
        setOperations([])
        setRaw('')
        setError('missing app, environment, or key')
        setLoading(false)
      }
      return
    }

    try {
      const [elementData, versionsData, operationsData] = await Promise.all([
        requestElement(appId, env, key),
        requestVersions(appId, env, key),
        requestOperations(appId, env, key),
      ])

      if (!mountedRef.current || requestId !== requestSeq.current) return

      setElement(elementData)
      setVersions(versionsData.elements || [])
      setOperations(operationsData.operations || [])
      setRaw(decodeRaw(elementData.raw))
      setDiffText('')
      setError('')
    } catch (err) {
      if (!mountedRef.current || requestId !== requestSeq.current) return
      setElement(null)
      setVersions([])
      setOperations([])
      setRaw('')
      setDiffText('')
      setError(getErrorMessage(err, 'failed to load element detail'))
    } finally {
      if (mountedRef.current && requestId === requestSeq.current) setLoading(false)
    }
  }, [appId, env, key])

  useEffect(() => {
    mountedRef.current = true

    queueMicrotask(() => {
      void loadPage()
    })

    return () => {
      mountedRef.current = false
    }
  }, [loadPage])

  async function handleSave() {
    const startedAppId = appId
    const startedEnv = env
    const startedKey = key
    if (!startedAppId || !startedEnv || !startedKey) return

    setSaving(true)
    setError('')

    try {
      await apiRequest<void>(
        `/api/apps/${encodeURIComponent(startedAppId)}/envs/${encodeURIComponent(startedEnv)}/elements/${encodeURIComponent(startedKey)}`,
        { ...jsonBody({ raw }), method: 'PUT' },
      )

      if (!canApplyMutationResult(startedAppId, startedEnv, startedKey)) return

      setLoading(true)
      await loadPage()
    } catch (err) {
      if (canApplyMutationResult(startedAppId, startedEnv, startedKey)) {
        setError(getErrorMessage(err, 'failed to update element content'))
      }
    } finally {
      if (mountedRef.current) setSaving(false)
    }
  }

  async function handleLoadDiff() {
    const startedAppId = appId
    const startedEnv = env
    const startedKey = key
    const base = diffBase.trim()
    const compare = diffCompare.trim()
    const requestId = ++diffRequestSeq.current

    if (!startedAppId || !startedEnv || !startedKey || !base || !compare) return

    setDiffLoading(true)
    setError('')

    try {
      const data = await apiRequest<DiffResponse>(
        `/api/apps/${encodeURIComponent(startedAppId)}/envs/${encodeURIComponent(startedEnv)}/elements/${encodeURIComponent(startedKey)}/diff${buildQuery({ base, compare })}`,
      )

      if (!mountedRef.current || requestId !== diffRequestSeq.current) return
      if (!canApplyMutationResult(startedAppId, startedEnv, startedKey)) return

      setDiffText(data.diff || '')
    } catch (err) {
      if (mountedRef.current && requestId === diffRequestSeq.current && canApplyMutationResult(startedAppId, startedEnv, startedKey)) {
        setDiffText('')
        setError(getErrorMessage(err, 'failed to load diff'))
      }
    } finally {
      if (mountedRef.current && requestId === diffRequestSeq.current) setDiffLoading(false)
    }
  }

  const metadata = element?.metadata

  return (
    <Stack spacing={3}>
      <Stack direction={{ xs: 'column', lg: 'row' }} spacing={3} alignItems="stretch">
        <Box sx={{ flex: 1 }}>
          <Stack direction="row" spacing={1} alignItems="center">
            <Inventory2Icon color="primary" />
            <Typography variant="h4" component="h1">
              Element detail
            </Typography>
          </Stack>
          <Typography color="text.secondary">App: {appId || 'unknown'} / Env: {env || 'unknown'} / Key: {key || 'unknown'}</Typography>
        </Box>
        <Paper variant="outlined" sx={{ p: 2, minWidth: { lg: 280 } }}>
          <Stack spacing={1.5}>
            <Typography variant="h6" component="h2">Status</Typography>
            <Typography>Latest: {metadata?.latestVersion ?? '-'}</Typography>
            <Typography>Using: {metadata?.usingVersion ?? '-'}</Typography>
            <Typography>Draft: {metadata?.unpublishedVersion ?? '-'}</Typography>
            <Typography>Type: {getContentTypeLabel(metadata?.contentType)}</Typography>
            <Divider />
            <Button component={RouterLink} to={`/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}/publish`} variant="outlined" startIcon={<PublishIcon />}>
              Publish
            </Button>
            <Button component={RouterLink} to={`/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}/rollback`} variant="outlined" startIcon={<LockResetIcon />}>
              Rollback
            </Button>
          </Stack>
        </Paper>
      </Stack>

      {error && <ErrorState message={error} />}

      {loading ? (
        <LoadingState label="Loading element detail" />
      ) : !element ? (
        <EmptyState title="Element not found" description="Reload the page or return to the elements list." />
      ) : (
        <Paper>
          <Tabs value={tab} onChange={(_, nextValue: number) => setTab(nextValue)} aria-label="element detail tabs" variant="scrollable" scrollButtons="auto">
            <Tab label="Content" />
            <Tab label="Versions" />
            <Tab label="Operations" />
            <Tab label="Instances" />
          </Tabs>
          <Divider />
          <Box sx={{ p: 3 }}>
            {tab === 0 && (
              <Stack spacing={2}>
                <Typography variant="h6" component="h2">Content</Typography>
                <TextField label="Raw" value={raw} onChange={(event) => setRaw(event.target.value)} fullWidth multiline minRows={12} disabled={saving} />
                <Stack direction="row" justifyContent="flex-end">
                  <Button variant="contained" startIcon={<SaveIcon />} onClick={() => void handleSave()} disabled={saving}>Save content</Button>
                </Stack>
              </Stack>
            )}

            {tab === 1 && (
              <Stack spacing={3}>
                <Typography variant="h6" component="h2">Versions</Typography>
                {versions.length === 0 ? (
                  <EmptyState title="No versions found" description="This element does not have any version history yet." />
                ) : (
                  <>
                    <TableContainer component={Paper} variant="outlined">
                      <Table>
                        <TableHead>
                          <TableRow>
                            <TableCell>Version</TableCell>
                            <TableCell>Published</TableCell>
                            <TableCell>Type</TableCell>
                            <TableCell>Preview</TableCell>
                          </TableRow>
                        </TableHead>
                        <TableBody>
                          {versions.map((version) => (
                            <TableRow key={`${version.metadata?.key || key}-${version.version}`} hover>
                              <TableCell>{version.version ?? '-'}</TableCell>
                              <TableCell>{version.published ? 'Yes' : 'No'}</TableCell>
                              <TableCell>{getContentTypeLabel(version.metadata?.contentType)}</TableCell>
                              <TableCell sx={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>{decodeRaw(version.raw).slice(0, 120) || '-'}</TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </TableContainer>

                    <Paper variant="outlined" sx={{ p: 2 }}>
                      <Stack spacing={2}>
                        <Typography variant="subtitle1">Compare versions</Typography>
                        <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                          <TextField label="Base" value={diffBase} onChange={(event) => setDiffBase(event.target.value)} fullWidth helperText="Older or current version number used as diff base." />
                          <TextField label="Compare" value={diffCompare} onChange={(event) => setDiffCompare(event.target.value)} fullWidth helperText="Version number to compare against the base version." />
                          <Button variant="outlined" startIcon={<CompareArrowsIcon />} onClick={() => void handleLoadDiff()} disabled={diffLoading || !diffBase.trim() || !diffCompare.trim()} sx={{ alignSelf: { md: 'center' } }}>
                            Show diff
                          </Button>
                        </Stack>
                        {diffLoading ? (
                          <LoadingState label="Loading diff" />
                        ) : diffText ? (
                          <TextField label="Diff" value={diffText} multiline minRows={10} fullWidth InputProps={{ readOnly: true }} />
                        ) : (
                          <Typography color="text.secondary">Select two versions to compare.</Typography>
                        )}
                      </Stack>
                    </Paper>
                  </>
                )}
              </Stack>
            )}

            {tab === 2 && (
              <Stack spacing={2}>
                <Stack direction="row" spacing={1} alignItems="center">
                  <TimelineIcon color="primary" fontSize="small" />
                  <Typography variant="h6" component="h2">Operations</Typography>
                </Stack>
                {operations.length === 0 ? (
                  <EmptyState title="No operations found" description="No recorded operations are available for this element yet." />
                ) : (
                  <TableContainer component={Paper} variant="outlined">
                    <Table>
                      <TableHead>
                        <TableRow>
                          <TableCell>Operator</TableCell>
                          <TableCell>Transition</TableCell>
                          <TableCell>At</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {operations.map((operation, index) => (
                          <TableRow key={`${operation.operatedAt || 0}-${operation.operator || 'unknown'}-${index}`} hover>
                            <TableCell>{operation.operator || '-'}</TableCell>
                            <TableCell>v{operation.lastVersion ?? '-'} → v{operation.currentVersion ?? '-'}</TableCell>
                            <TableCell>{formatTimestamp(operation.operatedAt)}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                )}
              </Stack>
            )}

            {tab === 3 && (
              <Stack spacing={2}>
                <Typography variant="h6" component="h2">Instances</Typography>
                <Typography color="text.secondary">View instances currently associated with this element in the cluster instances page.</Typography>
                <Stack direction="row">
                  <Button component={RouterLink} to={`/cluster/instances${buildQuery({ app: appId, env, key })}`} variant="outlined" startIcon={<LaunchIcon />}>
                    View instances
                  </Button>
                </Stack>
              </Stack>
            )}
          </Box>
        </Paper>
      )}
    </Stack>
  )
}
