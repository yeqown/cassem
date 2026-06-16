import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline'
import CompareArrowsIcon from '@mui/icons-material/CompareArrows'
import Inventory2Icon from '@mui/icons-material/Inventory2'
import LaunchIcon from '@mui/icons-material/Launch'
import LockResetIcon from '@mui/icons-material/LockReset'
import PublishIcon from '@mui/icons-material/Publish'
import RadioButtonUncheckedIcon from '@mui/icons-material/RadioButtonUnchecked'
import SaveIcon from '@mui/icons-material/Save'
import TimelineIcon from '@mui/icons-material/Timeline'
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined'
import {
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControl,
  IconButton,
  InputLabel,
  MenuItem,
  Paper,
  Select,
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
import { alpha } from '@mui/material/styles'
import { Link as RouterLink, useParams } from 'react-router-dom'
import { AppBreadcrumbs } from '../../components/AppBreadcrumbs'
import { EmptyState, ErrorState, LoadingState } from '../../components/StateView'
import { contentTypes, formatVersionLabel, type DiffResponse, type Element, type ElementOperation, type ElementOperationsResponse, type ElementsResponse } from '../../domain/types'
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

const operationLabels = new Map<string | number, string>([
  [1, 'SET'],
  [2, 'UNSET'],
  [3, 'PUBLISH'],
  ['1', 'SET'],
  ['2', 'UNSET'],
  ['3', 'PUBLISH'],
])

type DiffTone = 'added' | 'removed' | 'plain'

type DiffChunk = {
  text: string
  tone: DiffTone
}

function getOperationLabel(op?: string | number) {
  if (op === undefined || op === null || op === '') return '-'
  return operationLabels.get(op) || String(op)
}

function formatVersionChange(operation: ElementOperation) {
  const last = operation.lastVersion
  const current = operation.currentVersion
  if (!last || !current || last === current) return '-'
  return `v${last} → v${current}`
}

function parseAnsiDiff(value: string) {
  const chunks: DiffChunk[] = []
  const pattern = new RegExp(`${String.fromCharCode(27)}\\[(31|32|0)m`, 'g')
  let tone: DiffTone = 'plain'
  let cursor = 0

  for (const match of value.matchAll(pattern)) {
    if (match.index > cursor) {
      chunks.push({ text: value.slice(cursor, match.index), tone })
    }
    tone = match[1] === '31' ? 'removed' : match[1] === '32' ? 'added' : 'plain'
    cursor = match.index + match[0].length
  }

  if (cursor < value.length) {
    chunks.push({ text: value.slice(cursor), tone })
  }

  return chunks.filter((chunk) => chunk.text.length > 0)
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
  const [previewVersion, setPreviewVersion] = useState<Element | null>(null)
  const [diffBase, setDiffBase] = useState('')
  const [diffCompare, setDiffCompare] = useState('')
  const [diffText, setDiffText] = useState('')
  const requestSeq = useRef(0)
  const diffRequestSeq = useRef(0)
  const mountedRef = useRef(false)
  const lastLoadKeyRef = useRef('')
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
        setPreviewVersion(null)
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

      const nextVersions = versionsData.elements || []
      const versionOptions = nextVersions
        .map((version) => version.version)
        .filter((version): version is number => typeof version === 'number')

      setElement(elementData)
      setVersions(nextVersions)
      setOperations(operationsData.operations || [])
      setRaw(decodeRaw(elementData.raw))
      setPreviewVersion(null)
      setDiffBase(versionOptions[0] ? String(versionOptions[0]) : '')
      setDiffCompare(versionOptions.length > 1 ? String(versionOptions[versionOptions.length - 1]) : '')
      setDiffText('')
      setError('')
    } catch (err) {
      if (!mountedRef.current || requestId !== requestSeq.current) return
      setElement(null)
      setVersions([])
      setOperations([])
      setRaw('')
      setPreviewVersion(null)
      setDiffText('')
      setError(getErrorMessage(err, 'failed to load element detail'))
    } finally {
      if (mountedRef.current && requestId === requestSeq.current) setLoading(false)
    }
  }, [appId, env, key])

  useEffect(() => {
    mountedRef.current = true

    const loadKey = JSON.stringify({ appId, env, key })
    if (lastLoadKeyRef.current !== loadKey) {
      lastLoadKeyRef.current = loadKey
      queueMicrotask(() => {
        void loadPage()
      })
    }

    return () => {
      mountedRef.current = false
    }
  }, [appId, env, key, loadPage])

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
  const previewVersionLabel = previewVersion?.version === undefined || previewVersion.version === null ? '-' : `v${previewVersion.version}`
  const previewRaw = previewVersion ? decodeRaw(previewVersion.raw) : ''

  return (
    <Stack spacing={3}>
      <AppBreadcrumbs items={[
        { label: 'Apps', to: '/apps' },
        { label: appId || 'unknown', to: `/apps/${encodeURIComponent(appId)}/envs` },
        { label: env || 'unknown', to: `/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements` },
        { label: key || 'unknown', to: `/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}` },
        { label: 'Detail' },
      ]} />

      <Paper variant="outlined" sx={{ p: 2.5, borderRadius: 3 }}>
        <Stack direction={{ xs: 'column', lg: 'row' }} spacing={2} alignItems={{ xs: 'stretch', lg: 'flex-start' }}>
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Stack direction="row" spacing={1} alignItems="center">
              <Inventory2Icon color="primary" />
              <Typography variant="h4" component="h1">
                Element detail
              </Typography>
            </Stack>
            <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap" sx={{ mt: 1.5 }}>
              <Chip label={`Latest: ${formatVersionLabel(metadata?.latestVersion)}`} />
              <Chip label={`Current: ${formatVersionLabel(metadata?.usingVersion)}`} />
              <Chip label={`Draft: ${formatVersionLabel(metadata?.unpublishedVersion)}`} />
              <Chip label={`Type: ${getContentTypeLabel(metadata?.contentType)}`} />
            </Stack>
          </Box>
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5} data-testid="element-detail-actions" sx={{ alignSelf: 'flex-end' }}>
            <Button component={RouterLink} to={`/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}/publish`} variant="outlined" startIcon={<PublishIcon />}>
              Publish
            </Button>
            <Button component={RouterLink} to={`/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}/rollback`} variant="outlined" startIcon={<LockResetIcon />}>
              Rollback
            </Button>
          </Stack>
        </Stack>
      </Paper>

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
                            <TableCell>State</TableCell>
                            <TableCell>Type</TableCell>
                            <TableCell>Preview</TableCell>
                          </TableRow>
                        </TableHead>
                        <TableBody>
                          {versions.map((version) => {
                            const isCurrent = metadata?.usingVersion === version.version
                            const decodedRaw = decodeRaw(version.raw)
                            const versionLabel = version.version === undefined || version.version === null ? '-' : `v${version.version}`

                            return (
                              <TableRow
                                key={`${version.metadata?.key || key}-${version.version}`}
                                data-testid={`version-row-${version.version}`}
                                hover
                                sx={isCurrent
                                  ? {
                                      bgcolor: (theme) => alpha(theme.palette.primary.main, 0.08),
                                      boxShadow: (theme) => `inset 4px 0 0 ${theme.palette.primary.main}`,
                                      '&:hover': { bgcolor: (theme) => alpha(theme.palette.primary.main, 0.12) },
                                    }
                                  : undefined}
                              >
                                <TableCell>
                                  <Stack direction="row" spacing={0.75} alignItems="baseline">
                                    <Typography component="span" sx={{ color: isCurrent ? 'primary.main' : 'text.primary', fontWeight: isCurrent ? 700 : 400 }}>
                                      {versionLabel}
                                    </Typography>
                                    {isCurrent && (
                                      <Typography component="span" variant="caption" sx={{ color: 'primary.main', fontWeight: 600 }}>
                                        (current)
                                      </Typography>
                                    )}
                                  </Stack>
                                </TableCell>
                                <TableCell>
                                  <Stack direction="row" spacing={0.75} alignItems="center" sx={{ color: version.published ? 'success.main' : 'text.secondary', fontWeight: 600 }}>
                                    {version.published ? <CheckCircleOutlineIcon fontSize="small" /> : <RadioButtonUncheckedIcon fontSize="small" />}
                                    <Typography component="span" variant="body2" sx={{ fontWeight: 600 }}>
                                      {version.published ? 'Published' : 'Draft'}
                                    </Typography>
                                  </Stack>
                                </TableCell>
                                <TableCell>{getContentTypeLabel(version.metadata?.contentType)}</TableCell>
                                <TableCell sx={{ maxWidth: 420 }}>
                                  <Stack direction="row" spacing={1} alignItems="center" sx={{ minWidth: 0 }}>
                                    <Typography
                                      data-testid={`version-preview-${version.version}`}
                                      title={decodedRaw || '-'}
                                      sx={{ minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1 }}
                                    >
                                      {decodedRaw || '-'}
                                    </Typography>
                                    <IconButton size="small" aria-label={`Preview ${versionLabel}`} onClick={() => setPreviewVersion(version)}>
                                      <VisibilityOutlinedIcon fontSize="small" />
                                    </IconButton>
                                  </Stack>
                                </TableCell>
                              </TableRow>
                            )
                          })}
                        </TableBody>
                      </Table>
                    </TableContainer>

                    <Paper variant="outlined" sx={{ p: 2.5, borderRadius: 3 }}>
                      <Stack spacing={2}>
                        <Typography variant="subtitle1">Compare versions</Typography>
                        <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} alignItems={{ md: 'flex-start' }}>
                          <FormControl fullWidth>
                            <InputLabel id="diff-base-label">Base</InputLabel>
                            <Select labelId="diff-base-label" value={diffBase} label="Base" onChange={(event) => setDiffBase(event.target.value)}>
                              {versions.map((version) => (
                                <MenuItem key={`base-${version.version}`} value={String(version.version)}>
                                  v{version.version}
                                </MenuItem>
                              ))}
                            </Select>
                          </FormControl>
                          <FormControl fullWidth>
                            <InputLabel id="diff-compare-label">Compare</InputLabel>
                            <Select labelId="diff-compare-label" value={diffCompare} label="Compare" onChange={(event) => setDiffCompare(event.target.value)}>
                              {versions.map((version) => (
                                <MenuItem key={`compare-${version.version}`} value={String(version.version)}>
                                  v{version.version}
                                </MenuItem>
                              ))}
                            </Select>
                          </FormControl>
                          <Button variant="outlined" startIcon={<CompareArrowsIcon />} onClick={() => void handleLoadDiff()} disabled={diffLoading || !diffBase.trim() || !diffCompare.trim()} sx={{ minWidth: 140, whiteSpace: 'nowrap', alignSelf: { md: 'center' } }}>
                            Show diff
                          </Button>
                        </Stack>
                        {diffLoading ? (
                          <LoadingState label="Loading diff" />
                        ) : diffText ? (
                          <Box aria-label="Diff" component="pre" sx={{ m: 0, minHeight: 160, p: 2, border: 1, borderColor: 'divider', borderRadius: 2, bgcolor: 'background.default', whiteSpace: 'pre-wrap', wordBreak: 'break-word', fontFamily: 'monospace' }}>
                            {parseAnsiDiff(diffText).map((chunk, index) => (
                              <Box key={`${chunk.tone}-${index}`} component="span" sx={{ color: chunk.tone === 'removed' ? 'error.main' : chunk.tone === 'added' ? 'success.main' : 'text.primary', fontWeight: chunk.tone === 'plain' ? 400 : 700 }}>
                                {chunk.text}
                              </Box>
                            ))}
                          </Box>
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
                          <TableCell>Time</TableCell>
                          <TableCell>Operation</TableCell>
                          <TableCell>Operator</TableCell>
                          <TableCell>Version change</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {operations.map((operation, index) => (
                          <TableRow key={`${operation.operatedAt || 0}-${operation.operator || 'unknown'}-${index}`} hover>
                            <TableCell>{formatTimestamp(operation.operatedAt)}</TableCell>
                            <TableCell>{getOperationLabel(operation.op)}</TableCell>
                            <TableCell>{operation.operator || '-'}</TableCell>
                            <TableCell>{formatVersionChange(operation)}</TableCell>
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

      <Dialog open={Boolean(previewVersion)} onClose={() => setPreviewVersion(null)} fullWidth maxWidth="md" aria-labelledby="version-preview-title">
        <DialogTitle id="version-preview-title">Version {previewVersionLabel} Preview</DialogTitle>
        <DialogContent>
          <Box component="pre" sx={{ m: 0, p: 2, border: 1, borderColor: 'divider', borderRadius: 2, bgcolor: 'background.default', whiteSpace: 'pre-wrap', wordBreak: 'break-word', fontFamily: 'monospace' }}>
            {previewRaw || '-'}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setPreviewVersion(null)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Stack>
  )
}
