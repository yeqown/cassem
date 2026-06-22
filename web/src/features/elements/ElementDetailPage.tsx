import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline'
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline'
import CompareArrowsIcon from '@mui/icons-material/CompareArrows'
import Inventory2Icon from '@mui/icons-material/Inventory2'
import LaunchIcon from '@mui/icons-material/Launch'
import LockResetIcon from '@mui/icons-material/LockReset'
import PublishIcon from '@mui/icons-material/Publish'
import RadioButtonUncheckedIcon from '@mui/icons-material/RadioButtonUnchecked'
import SaveIcon from '@mui/icons-material/Save'
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined'
import {
  Alert,
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
  Typography,
} from '@mui/material'
import { alpha } from '@mui/material/styles'
import { Link as RouterLink, useParams } from 'react-router-dom'
import { AppBreadcrumbs } from '../../components/AppBreadcrumbs'
import { DangerConfirmDialog } from '../../components/DangerConfirmDialog'
import { DiffViewer } from '../../components/DiffViewer'
import { EmptyState, ErrorState, LoadingState } from '../../components/StateView'
import { useToast } from '../../components/ToastProvider'
import { useErrorState } from '../../components/useErrorState'
import { formatVersionLabel, type Element, type ElementOperation, type ElementOperationsResponse, type ElementsResponse, type RetentionPolicy } from '../../domain/types'
import { ApiError, apiRequest, buildQuery, jsonBody } from '../../lib/api'
import { decodeRaw } from '../../lib/raw'
import { readSettings } from '../../lib/settings'
import { ContentEditor } from './ContentEditor'
import { ContentViewer } from './ContentViewer'
import { validateContent, type ContentValidationResult } from './contentValidation'
import { getContentTypeLabel } from './contentView'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback
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

async function requestRetentionPolicy() {
  return apiRequest<RetentionPolicy>('/api/admin/retention')
}

export function ElementDetailPage() {
  const { appId = '', env = '', key = '' } = useParams()
  const { showToast } = useToast()
  const [element, setElement] = useState<Element | null>(null)
  const [versions, setVersions] = useState<Element[]>([])
  const [operations, setOperations] = useState<ElementOperation[]>([])
  const [retentionPolicy, setRetentionPolicy] = useState<RetentionPolicy | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [diffLoaded, setDiffLoaded] = useState(false)
  const [error, setError] = useErrorState()
  const [tab, setTab] = useState(0)
  const [settings] = useState(readSettings)
  const [raw, setRaw] = useState('')
  const [editRaw, setEditRaw] = useState('')
  const [editValidation, setEditValidation] = useState<ContentValidationResult>({ valid: true })
  const [newVersionOpen, setNewVersionOpen] = useState(false)
  const [publishDirectConfirmOpen, setPublishDirectConfirmOpen] = useState(false)
  const [previewVersion, setPreviewVersion] = useState<Element | null>(null)
  const [diffBase, setDiffBase] = useState('')
  const [diffCompare, setDiffCompare] = useState('')
  const [diffPair, setDiffPair] = useState<{ base: Element; compare: Element } | null>(null)
  const requestSeq = useRef(0)
  const mountedRef = useRef(false)
  const lastLoadKeyRef = useRef('')
  const locationRef = useRef({ appId, env, key })

  useLayoutEffect(() => {
    locationRef.current = { appId, env, key }
  }, [appId, env, key, setError])

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
        setRetentionPolicy(null)
        setRaw('')
        setEditRaw('')
        setEditValidation({ valid: true })
        setNewVersionOpen(false)
        setPublishDirectConfirmOpen(false)
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
      setRetentionPolicy(null)
      setRaw(decodeRaw(elementData.raw))
      setEditRaw(decodeRaw(elementData.raw))
      setEditValidation({ valid: true })
      setNewVersionOpen(false)
      setPreviewVersion(null)
      setDiffBase(versionOptions[0] ? String(versionOptions[0]) : '')
      setDiffCompare(versionOptions.length > 1 ? String(versionOptions[versionOptions.length - 1]) : '')
      setDiffPair(null)
      setDiffLoaded(false)
      setError('')
      void requestRetentionPolicy()
        .then((policy) => {
          if (mountedRef.current && requestId === requestSeq.current) setRetentionPolicy(policy)
        })
        .catch(() => {
          if (mountedRef.current && requestId === requestSeq.current) setRetentionPolicy(null)
        })
    } catch (err) {
      if (!mountedRef.current || requestId !== requestSeq.current) return
      setElement(null)
      setVersions([])
      setOperations([])
      setRetentionPolicy(null)
      setRaw('')
      setEditRaw('')
      setNewVersionOpen(false)
      setPreviewVersion(null)
      setDiffPair(null)
      setDiffLoaded(false)
      setError(getErrorMessage(err, 'failed to load element detail'))
    } finally {
      if (mountedRef.current && requestId === requestSeq.current) setLoading(false)
    }
  }, [appId, env, key, setError])

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

  useEffect(() => {
    if (!newVersionOpen) return

    const timer = window.setTimeout(() => {
      setEditValidation(validateContent(element?.metadata?.contentType, editRaw))
    }, 250)

    return () => window.clearTimeout(timer)
  }, [newVersionOpen, element?.metadata?.contentType, editRaw])

  function handleOpenNewVersion() {
    const draftVersion = metadata?.unpublishedVersion
    if (draftVersion) {
      setError(`Draft v${draftVersion} already exists. Publish or rollback it before creating a new version.`)
      return
    }

    setEditRaw(raw)
    setEditValidation(validateContent(element?.metadata?.contentType, raw))
    setError('')
    setNewVersionOpen(true)
  }

  function closeNewVersionDialog() {
    if (saving) return
    setNewVersionOpen(false)
    setEditValidation({ valid: true })
  }

  async function handleSubmitNewVersion() {
    const startedAppId = appId
    const startedEnv = env
    const startedKey = key
    if (!startedAppId || !startedEnv || !startedKey) return

    const validation = validateContent(element?.metadata?.contentType, editRaw)
    setEditValidation(validation)
    if (!validation.valid) return

    setSaving(true)
    setError('')

    try {
      await apiRequest<void>(
        `/api/apps/${encodeURIComponent(startedAppId)}/envs/${encodeURIComponent(startedEnv)}/elements/${encodeURIComponent(startedKey)}`,
        { ...jsonBody({ raw: editRaw }), method: 'PUT' },
      )

      if (!canApplyMutationResult(startedAppId, startedEnv, startedKey)) return

      setNewVersionOpen(false)
      setEditValidation({ valid: true })
      setLoading(true)
      await loadPage()
    } catch (err) {
      if (canApplyMutationResult(startedAppId, startedEnv, startedKey)) {
        setError(getErrorMessage(err, 'failed to create new version'))
      }
    } finally {
      if (mountedRef.current) setSaving(false)
    }
  }

  async function handlePublishDirectly() {
    const startedAppId = appId
    const startedEnv = env
    const startedKey = key
    if (!startedAppId || !startedEnv || !startedKey) return

    const validation = validateContent(element?.metadata?.contentType, editRaw)
    setEditValidation(validation)
    if (!validation.valid) return

    setSaving(true)
    setError('')

    try {
      await apiRequest<void>(
        `/api/apps/${encodeURIComponent(startedAppId)}/envs/${encodeURIComponent(startedEnv)}/elements/${encodeURIComponent(startedKey)}`,
        { ...jsonBody({ raw: editRaw }), method: 'PUT' },
      )

      if (!canApplyMutationResult(startedAppId, startedEnv, startedKey)) return

      const updatedElement = await requestElement(startedAppId, startedEnv, startedKey)
      if (!canApplyMutationResult(startedAppId, startedEnv, startedKey)) return

      const previousLatestVersion = metadata?.latestVersion || 0
      const latestVersion = updatedElement.metadata?.latestVersion || 0
      const version = updatedElement.metadata?.unpublishedVersion || (latestVersion > previousLatestVersion ? latestVersion : 0)
      if (!version) throw new Error('new version was created without a publishable draft')

      await apiRequest<void>(
        `/api/apps/${encodeURIComponent(startedAppId)}/envs/${encodeURIComponent(startedEnv)}/elements/${encodeURIComponent(startedKey)}/publish`,
        jsonBody({ version, publishMode: 2 }),
      )

      if (!canApplyMutationResult(startedAppId, startedEnv, startedKey)) return

      setPublishDirectConfirmOpen(false)
      setNewVersionOpen(false)
      setEditValidation({ valid: true })
      showToast(`Version ${version} was queued for full publish.`, 'success')
      setLoading(true)
      await loadPage()
    } catch (err) {
      if (canApplyMutationResult(startedAppId, startedEnv, startedKey)) {
        setError(getErrorMessage(err, 'failed to publish directly'))
      }
    } finally {
      if (mountedRef.current) setSaving(false)
    }
  }

  function handleLoadDiff() {
    const base = Number(diffBase.trim())
    const compare = Number(diffCompare.trim())

    if (!Number.isFinite(base) || !Number.isFinite(compare) || base <= 0 || compare <= 0) return

    const baseElement = versions.find((version) => version.version === base)
    const compareElement = versions.find((version) => version.version === compare)

    if (!baseElement || !compareElement) {
      setDiffPair(null)
      setDiffLoaded(false)
      setError('selected versions are not available in the loaded version list')
      return
    }

    setDiffPair({ base: baseElement, compare: compareElement })
    setDiffLoaded(true)
    setError('')
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

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} justifyContent="space-between" alignItems={{ md: 'center' }}>
        <Box sx={{ minWidth: 0 }}>
          <Stack direction="row" spacing={1} alignItems="center">
            <Inventory2Icon color="primary" />
            <Typography variant="h4" component="h1">
              Element detail
            </Typography>
          </Stack>
          <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap" sx={{ mt: 1 }}>
            <Chip label={`Latest: ${formatVersionLabel(metadata?.latestVersion)}`} />
            <Chip label={`Current: ${formatVersionLabel(metadata?.usingVersion)}`} />
            <Chip label={`Draft: ${formatVersionLabel(metadata?.unpublishedVersion)}`} />
            <Chip label={`Type: ${getContentTypeLabel(metadata?.contentType)}`} />
          </Stack>
        </Box>
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} data-testid="element-detail-actions">
          <Button variant="outlined" startIcon={<AddCircleOutlineIcon />} onClick={handleOpenNewVersion}>
            New Version
          </Button>
          <Button component={RouterLink} to={`/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}/publish`} variant="outlined" startIcon={<PublishIcon />}>
            Publish
          </Button>
          <Button component={RouterLink} to={`/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}/rollback`} variant="outlined" startIcon={<LockResetIcon />}>
            Rollback
          </Button>
        </Stack>
      </Stack>

      {error.message && <ErrorState message={error.message} eventKey={error.eventKey} />}

      {loading ? (
        <LoadingState label="Loading element detail" />
      ) : !element ? (
        <EmptyState title="Element not found" description="Reload the page or return to the elements list." />
      ) : (
        <Stack spacing={3}>
          <Paper>
            <Stack direction="row" spacing={1} alignItems="center" justifyContent="space-between" sx={{ px: 3, py: 2, borderBottom: 1, borderColor: 'divider' }}>
              <Typography variant="h6" component="h2">Content</Typography>
              <Chip size="small" label={`${getContentTypeLabel(metadata?.contentType)} · read-only`} />
            </Stack>
            <ContentViewer value={raw} contentType={metadata?.contentType} ariaLabel="Current content" bordered={false} codeTheme={settings.codeTheme} lineWrapping={settings.editorLineWrapping} minRows={8} showLabel={false} />
          </Paper>

          <Paper>
            <Tabs value={tab} onChange={(_, nextValue: number) => setTab(nextValue)} aria-label="element detail tabs" variant="scrollable" scrollButtons="auto">
              <Tab label="Versions" />
              <Tab label="Diff" />
              <Tab label="Operations" />
              <Tab label="Instances" />
            </Tabs>
            <Divider />
            <Box sx={{ p: 3 }}>
            {tab === 0 && (
              <Stack spacing={3}>
                {retentionPolicy?.enabled && retentionPolicy.versionPolicy && (
                  <Alert severity="info" variant="outlined">{retentionPolicy.versionPolicy}</Alert>
                )}
                {versions.length === 0 ? (
                  <EmptyState title="No versions found" description="This element does not have any version history yet." />
                ) : (
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
                )}
              </Stack>
            )}

            {tab === 1 && (
              <Stack spacing={2}>
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
                  <Button variant="outlined" startIcon={<CompareArrowsIcon />} onClick={handleLoadDiff} disabled={!diffBase.trim() || !diffCompare.trim()} sx={{ minWidth: 140, whiteSpace: 'nowrap', alignSelf: { md: 'center' } }}>
                    Show diff
                  </Button>
                </Stack>
                {diffLoaded && diffPair ? (
                  <DiffViewer
                    oldValue={decodeRaw(diffPair.base.raw)}
                    newValue={decodeRaw(diffPair.compare.raw)}
                    baseLabel={`Base v${diffBase}`}
                    compareLabel={`Compare v${diffCompare}`}
                  />
                ) : (
                  <Typography color="text.secondary">Select two versions to compare.</Typography>
                )}
              </Stack>
            )}

            {tab === 2 && (
              <Stack spacing={2}>
                {retentionPolicy?.enabled && retentionPolicy.operationPolicy && (
                  <Alert severity="info" variant="outlined">{retentionPolicy.operationPolicy}</Alert>
                )}
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
        </Stack>
      )}

      <Dialog open={newVersionOpen} onClose={closeNewVersionDialog} fullWidth maxWidth="md" aria-labelledby="new-version-title">
        <DialogTitle id="new-version-title">New Version</DialogTitle>
        <DialogContent>
          <ContentEditor value={editRaw} contentType={metadata?.contentType} ariaLabel="New version content" codeTheme={settings.codeTheme} disabled={saving} lineWrapping={settings.editorLineWrapping} validation={editValidation} onChange={setEditRaw} />
        </DialogContent>
        <DialogActions>
          <Button onClick={closeNewVersionDialog} disabled={saving}>Cancel</Button>
          <Button variant="outlined" startIcon={<PublishIcon />} onClick={() => setPublishDirectConfirmOpen(true)} disabled={saving || !editValidation.valid}>Publish directly</Button>
          <Button variant="contained" startIcon={<SaveIcon />} onClick={() => void handleSubmitNewVersion()} disabled={saving || !editValidation.valid}>Submit</Button>
        </DialogActions>
      </Dialog>

      <DangerConfirmDialog
        open={publishDirectConfirmOpen}
        title="Publish directly"
        description={<>This will create a new version and full publish it to all clients.</>}
        confirmLabel="Publish directly"
        loading={saving}
        onClose={() => setPublishDirectConfirmOpen(false)}
        onConfirm={() => void handlePublishDirectly()}
      />

      <Dialog open={Boolean(previewVersion)} onClose={() => setPreviewVersion(null)} fullWidth maxWidth="md" aria-labelledby="version-preview-title">
        <DialogTitle id="version-preview-title">Version {previewVersionLabel} Preview</DialogTitle>
        <DialogContent>
          <ContentViewer value={previewRaw} contentType={previewVersion?.metadata?.contentType} ariaLabel="Version preview content" codeTheme={settings.codeTheme} lineWrapping={settings.editorLineWrapping} />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setPreviewVersion(null)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Stack>
  )
}
