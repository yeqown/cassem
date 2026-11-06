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
import { useI18n } from '../../i18n'
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
  const { t, formatDateTime, formatNumber } = useI18n()
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
        setError(t('elements.error.missingDetailLocation'))
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
      setError(getErrorMessage(err, t('elements.error.loadDetail')))
    } finally {
      if (mountedRef.current && requestId === requestSeq.current) setLoading(false)
    }
  }, [appId, env, key, setError, t])

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
      setEditValidation(validateContent(element?.metadata?.contentType, editRaw, t))
    }, 250)

    return () => window.clearTimeout(timer)
  }, [newVersionOpen, element?.metadata?.contentType, editRaw, t])

  function handleOpenNewVersion() {
    const draftVersion = metadata?.unpublishedVersion
    if (draftVersion) {
      setError(t('elements.error.existingDraft', { version: draftVersion }))
      return
    }

    setEditRaw(raw)
    setEditValidation(validateContent(element?.metadata?.contentType, raw, t))
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

    const validation = validateContent(element?.metadata?.contentType, editRaw, t)
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
        setError(getErrorMessage(err, t('elements.error.createVersion')))
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

    const validation = validateContent(element?.metadata?.contentType, editRaw, t)
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
      if (!version) throw new Error(t('elements.error.missingPublishableDraft'))

      await apiRequest<void>(
        `/api/apps/${encodeURIComponent(startedAppId)}/envs/${encodeURIComponent(startedEnv)}/elements/${encodeURIComponent(startedKey)}/publish`,
        jsonBody({ version, publishMode: 2 }),
      )

      if (!canApplyMutationResult(startedAppId, startedEnv, startedKey)) return

      setPublishDirectConfirmOpen(false)
      setNewVersionOpen(false)
      setEditValidation({ valid: true })
      showToast(t('elements.publishQueued', { version }), 'success')
      setLoading(true)
      await loadPage()
    } catch (err) {
      if (canApplyMutationResult(startedAppId, startedEnv, startedKey)) {
        setError(getErrorMessage(err, t('elements.error.publishDirectly')))
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
      setError(t('elements.error.unavailableVersions'))
      return
    }

    setDiffPair({ base: baseElement, compare: compareElement })
    setDiffLoaded(true)
    setError('')
  }

  const metadata = element?.metadata
  const previewVersionLabel = previewVersion?.version === undefined || previewVersion.version === null ? '-' : `v${previewVersion.version}`
  const previewRaw = previewVersion ? decodeRaw(previewVersion.raw) : ''

  function formatTimestamp(timestamp?: number) {
    if (!timestamp) return '-'
    const milliseconds = timestamp > 1_000_000_000_000_000 ? Math.floor(timestamp / 1_000_000) : timestamp
    return formatDateTime(milliseconds)
  }

  return (
    <Stack spacing={3}>
      <AppBreadcrumbs items={[
        { label: t('elements.apps'), to: '/apps' },
        { label: appId || t('elements.unknown'), to: `/apps/${encodeURIComponent(appId)}/envs` },
        { label: env || t('elements.unknown'), to: `/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements` },
        { label: key || t('elements.unknown'), to: `/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}` },
        { label: t('elements.detail') },
      ]} />

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} justifyContent="space-between" alignItems={{ md: 'center' }}>
        <Box sx={{ minWidth: 0 }}>
          <Stack direction="row" spacing={1} alignItems="center">
            <Inventory2Icon color="primary" />
            <Typography variant="h4" component="h1">
              {t('elements.detailTitle')}
            </Typography>
          </Stack>
          <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap" sx={{ mt: 1 }}>
            <Chip label={t('elements.latestVersion', { version: formatVersionLabel(metadata?.latestVersion) })} />
            <Chip label={t('elements.currentVersion', { version: formatVersionLabel(metadata?.usingVersion) })} />
            <Chip label={t('elements.draftVersion', { version: formatVersionLabel(metadata?.unpublishedVersion) })} />
            <Chip label={t('elements.typeLabel', { type: getContentTypeLabel(metadata?.contentType) })} />
          </Stack>
        </Box>
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} data-testid="element-detail-actions">
          <Button variant="outlined" startIcon={<AddCircleOutlineIcon />} onClick={handleOpenNewVersion}>
            {t('elements.newVersion')}
          </Button>
          <Button component={RouterLink} to={`/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}/publish`} variant="outlined" startIcon={<PublishIcon />}>
            {t('elements.publish')}
          </Button>
          <Button component={RouterLink} to={`/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}/rollback`} variant="outlined" startIcon={<LockResetIcon />}>
            {t('elements.rollback')}
          </Button>
        </Stack>
      </Stack>

      {error.message && <ErrorState message={error.message} eventKey={error.eventKey} />}

      {loading ? (
        <LoadingState label={t('elements.loadingDetail')} />
      ) : !element ? (
        <EmptyState title={t('elements.notFound')} description={t('elements.notFoundDescription')} />
      ) : (
        <Stack spacing={3}>
          <Paper>
            <Stack direction="row" spacing={1} alignItems="center" justifyContent="space-between" sx={{ px: 3, py: 2, borderBottom: 1, borderColor: 'divider' }}>
              <Typography variant="h6" component="h2">{t('elements.content')}</Typography>
              <Chip size="small" label={t('elements.readOnlyType', { type: getContentTypeLabel(metadata?.contentType) })} />
            </Stack>
            <ContentViewer value={raw} contentType={metadata?.contentType} ariaLabel={t('elements.currentContent')} bordered={false} codeTheme={settings.codeTheme} lineWrapping={settings.editorLineWrapping} minRows={8} showLabel={false} />
          </Paper>

          <Paper>
            <Tabs value={tab} onChange={(_, nextValue: number) => setTab(nextValue)} aria-label={t('elements.detailTabs')} variant="scrollable" scrollButtons="auto">
              <Tab label={t('elements.versions')} />
              <Tab label={t('elements.diff')} />
              <Tab label={t('elements.operations')} />
              <Tab label={t('elements.instances')} />
            </Tabs>
            <Divider />
            <Box sx={{ p: 3 }}>
            {tab === 0 && (
              <Stack spacing={3}>
                {retentionPolicy?.enabled && retentionPolicy.versionPolicy && (
                  <Alert severity="info" variant="outlined">{t('elements.versionRetention', { count: formatNumber(retentionPolicy.keepVersionCount), days: formatNumber(retentionPolicy.keepVersionDays) })}</Alert>
                )}
                {versions.length === 0 ? (
                  <EmptyState title={t('elements.noVersions')} description={t('elements.noVersionsDescription')} />
                ) : (
                  <TableContainer component={Paper} variant="outlined">
                    <Table>
                      <TableHead>
                        <TableRow>
                          <TableCell>{t('elements.version')}</TableCell>
                          <TableCell>{t('elements.state')}</TableCell>
                          <TableCell>{t('elements.type')}</TableCell>
                          <TableCell>{t('elements.preview')}</TableCell>
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
                                      {t('elements.current')}
                                    </Typography>
                                  )}
                                </Stack>
                              </TableCell>
                              <TableCell>
                                <Stack direction="row" spacing={0.75} alignItems="center" sx={{ color: version.published ? 'success.main' : 'text.secondary', fontWeight: 600 }}>
                                  {version.published ? <CheckCircleOutlineIcon fontSize="small" /> : <RadioButtonUncheckedIcon fontSize="small" />}
                                  <Typography component="span" variant="body2" sx={{ fontWeight: 600 }}>
                                    {version.published ? t('elements.published') : t('elements.draft')}
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
                                  <IconButton size="small" aria-label={t('elements.previewVersion', { version: versionLabel })} onClick={() => setPreviewVersion(version)}>
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
                    <InputLabel id="diff-base-label">{t('elements.base')}</InputLabel>
                    <Select labelId="diff-base-label" value={diffBase} label={t('elements.base')} onChange={(event) => setDiffBase(event.target.value)}>
                      {versions.map((version) => (
                        <MenuItem key={`base-${version.version}`} value={String(version.version)}>
                          v{version.version}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                  <FormControl fullWidth>
                    <InputLabel id="diff-compare-label">{t('elements.compare')}</InputLabel>
                    <Select labelId="diff-compare-label" value={diffCompare} label={t('elements.compare')} onChange={(event) => setDiffCompare(event.target.value)}>
                      {versions.map((version) => (
                        <MenuItem key={`compare-${version.version}`} value={String(version.version)}>
                          v{version.version}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                  <Button variant="outlined" startIcon={<CompareArrowsIcon />} onClick={handleLoadDiff} disabled={!diffBase.trim() || !diffCompare.trim()} sx={{ minWidth: 140, whiteSpace: 'nowrap', alignSelf: { md: 'center' } }}>
                    {t('elements.showDiff')}
                  </Button>
                </Stack>
                {diffLoaded && diffPair ? (
                  <DiffViewer
                    oldValue={decodeRaw(diffPair.base.raw)}
                    newValue={decodeRaw(diffPair.compare.raw)}
                    baseLabel={t('elements.baseVersion', { version: diffBase })}
                    compareLabel={t('elements.compareVersion', { version: diffCompare })}
                    contentType={diffPair.compare.metadata?.contentType}
                  />
                ) : (
                  <Typography color="text.secondary">{t('elements.selectVersions')}</Typography>
                )}
              </Stack>
            )}

            {tab === 2 && (
              <Stack spacing={2}>
                {retentionPolicy?.enabled && retentionPolicy.operationPolicy && (
                  <Alert severity="info" variant="outlined">{t('elements.operationRetention', { days: formatNumber(retentionPolicy.keepOperationDays) })}</Alert>
                )}
                {operations.length === 0 ? (
                  <EmptyState title={t('elements.noOperations')} description={t('elements.noOperationsDescription')} />
                ) : (
                  <TableContainer component={Paper} variant="outlined">
                    <Table>
                      <TableHead>
                        <TableRow>
                          <TableCell>{t('elements.time')}</TableCell>
                          <TableCell>{t('elements.operation')}</TableCell>
                          <TableCell>{t('elements.operator')}</TableCell>
                          <TableCell>{t('elements.versionChange')}</TableCell>
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
                <Typography color="text.secondary">{t('elements.instancesDescription')}</Typography>
                <Stack direction="row">
                  <Button component={RouterLink} to={`/cluster/instances${buildQuery({ app: appId, env, key })}`} variant="outlined" startIcon={<LaunchIcon />}>
                    {t('elements.viewInstances')}
                  </Button>
                </Stack>
              </Stack>
            )}
            </Box>
          </Paper>
        </Stack>
      )}

      <Dialog open={newVersionOpen} onClose={closeNewVersionDialog} fullWidth maxWidth="md" aria-labelledby="new-version-title">
        <DialogTitle id="new-version-title">{t('elements.newVersion')}</DialogTitle>
        <DialogContent>
          <ContentEditor value={editRaw} contentType={metadata?.contentType} ariaLabel={t('elements.newVersionContent')} codeTheme={settings.codeTheme} disabled={saving} lineWrapping={settings.editorLineWrapping} validation={editValidation} onChange={setEditRaw} />
        </DialogContent>
        <DialogActions>
          <Button onClick={closeNewVersionDialog} disabled={saving}>{t('elements.cancel')}</Button>
          <Button variant="outlined" startIcon={<PublishIcon />} onClick={() => setPublishDirectConfirmOpen(true)} disabled={saving || !editValidation.valid}>{t('elements.publishDirectly')}</Button>
          <Button variant="contained" startIcon={<SaveIcon />} onClick={() => void handleSubmitNewVersion()} disabled={saving || !editValidation.valid}>{t('elements.submit')}</Button>
        </DialogActions>
      </Dialog>

      <DangerConfirmDialog
        open={publishDirectConfirmOpen}
        title={t('elements.publishDirectly')}
        description={t('elements.publishDirectDescription')}
        confirmLabel={t('elements.publishDirectly')}
        loading={saving}
        onClose={() => setPublishDirectConfirmOpen(false)}
        onConfirm={() => void handlePublishDirectly()}
      />

      <Dialog open={Boolean(previewVersion)} onClose={() => setPreviewVersion(null)} fullWidth maxWidth="md" aria-labelledby="version-preview-title">
        <DialogTitle id="version-preview-title">{t('elements.previewTitle', { version: previewVersionLabel })}</DialogTitle>
        <DialogContent>
          <ContentViewer value={previewRaw} contentType={previewVersion?.metadata?.contentType} ariaLabel={t('elements.previewContent')} codeTheme={settings.codeTheme} lineWrapping={settings.editorLineWrapping} />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setPreviewVersion(null)}>{t('elements.close')}</Button>
        </DialogActions>
      </Dialog>
    </Stack>
  )
}
