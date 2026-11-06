import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline'
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline'
import WarningAmberIcon from '@mui/icons-material/WarningAmber'
import {
  Box,
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControl,
  FormControlLabel,
  FormHelperText,
  InputLabel,
  LinearProgress,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  MenuItem,
  Paper,
  Select,
  Stack,
  Step,
  StepLabel,
  Stepper,
  Switch,
  Tab,
  Table,
  TableBody,
  Tabs,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material'
import { useNavigate } from 'react-router-dom'
import { useToast } from '../../components/ToastProvider'
import { contentTypes, type Element, type ElementsResponse } from '../../domain/types'
import { useI18n } from '../../i18n'
import { ApiError, apiRequest, buildQuery, jsonBody } from '../../lib/api'
import { decodeRaw } from '../../lib/raw'

const elementPageLimit = 100
const workerCount = 4

type EmptyElementStrategy = 'skip' | 'copy'
type CopyStep = 'create' | 'execute'
type ProgressState = 'pending' | 'doing' | 'done' | 'failed'
type CopyResultStatus = 'success' | 'skipped' | 'failed'

const copyResultStatuses: CopyResultStatus[] = ['success', 'skipped', 'failed']

type CopyResult = {
  key: string
  status: CopyResultStatus
  message?: string
  messageKey?: 'management.copy.emptySkipped' | 'management.copy.elementFailed'
}

type CopyProgress = {
  createEnv: ProgressState
  copyElements: ProgressState
  complete: ProgressState
  completed: number
  total: number
  results: CopyResult[]
}

type CopyEnvDialogProps = {
  open: boolean
  appId: string
  envs: string[]
  onClose: () => void
  onBusyChange: (busy: boolean) => void
  onFinished: () => void | Promise<void>
}

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback
}

function getElementKey(element: Element) {
  return element.metadata.key
}

function getContentTypeLabel(contentType?: number | string) {
  const value = Number(contentType)
  return contentTypes.find((item) => item.value === value)?.label || String(contentType || '-')
}

function isElementEmpty(element: Element) {
  return !element.metadata.usingVersion || element.raw === ''
}

async function requestElements(appId: string, env: string, seek = '') {
  return apiRequest<ElementsResponse>(
    `/api/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements${buildQuery({ limit: elementPageLimit, seek: seek || undefined })}`,
  )
}

async function requestAllElements(appId: string, env: string) {
  const elements: Element[] = []
  let seek = ''

  for (;;) {
    const data = await requestElements(appId, env, seek)
    elements.push(...(data.elements || []))
    if (!data.hasMore || !data.nextSeek) return elements
    seek = data.nextSeek
  }
}

async function requestElement(appId: string, env: string, key: string) {
  return apiRequest<Element>(`/api/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}`)
}

function normalizeContentType(contentType: Element['metadata']['contentType']) {
  if (typeof contentType === 'number') return contentType

  if (typeof contentType === 'string') {
    const matched = contentTypes.find((option) => option.label === contentType.toUpperCase())
    if (matched) return matched.value
    return contentType
  }

  return contentTypes[0].value
}

async function createEnv(appId: string, env: string) {
  return apiRequest<void>(`/api/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}`, { method: 'POST' })
}

async function createElement(appId: string, env: string, key: string, element: Element) {
  return apiRequest<void>(
    `/api/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}`,
    jsonBody({ raw: decodeRaw(element.raw || ''), contentType: normalizeContentType(element.metadata.contentType) }),
  )
}

function createInitialProgress(total = 0): CopyProgress {
  return {
    createEnv: 'pending',
    copyElements: 'pending',
    complete: 'pending',
    completed: 0,
    total,
    results: [],
  }
}

const copyResultStatusMeta = {
  success: { labelKey: 'management.copy.success', color: 'rgb(46, 125, 50)', backgroundColor: 'rgba(46, 125, 50, 0.08)' },
  skipped: { labelKey: 'management.copy.skipped', color: 'rgb(97, 97, 97)', backgroundColor: 'rgba(97, 97, 97, 0.08)' },
  failed: { labelKey: 'management.copy.failed', color: 'rgb(211, 47, 47)', backgroundColor: 'rgba(211, 47, 47, 0.08)' },
} as const

function getCopyResultIcon(status: CopyResultStatus) {
  if (status === 'success') return <CheckCircleOutlineIcon fontSize="small" aria-hidden="true" />
  if (status === 'skipped') return <WarningAmberIcon fontSize="small" aria-hidden="true" />
  return <ErrorOutlineIcon fontSize="small" aria-hidden="true" />
}

function CopyResultCount({ status, count, testId }: { status: CopyResultStatus; count: number; testId: string }) {
  const { t, formatNumber } = useI18n()
  const meta = copyResultStatusMeta[status]

  return (
    <Stack component="span" data-testid={testId} direction="row" spacing={0.5} alignItems="center" sx={{ color: meta.color, fontWeight: 700 }}>
      {getCopyResultIcon(status)}
      <Typography component="span" variant="body2" sx={{ color: 'inherit', fontWeight: 700 }}>
        {t('management.copy.resultCount', { status: t(meta.labelKey), count: formatNumber(count) })}
      </Typography>
    </Stack>
  )
}

function CopyResultState({ status }: { status: CopyResultStatus }) {
  const { t } = useI18n()
  const meta = copyResultStatusMeta[status]

  return (
    <Stack component="span" direction="row" spacing={0.5} alignItems="center" sx={{ color: meta.color, fontWeight: 700 }}>
      {getCopyResultIcon(status)}
      <Typography component="span" sx={{ color: 'inherit', fontSize: 'inherit', fontWeight: 700, lineHeight: 'inherit' }}>
        {t(meta.labelKey)}
      </Typography>
    </Stack>
  )
}

export function CopyEnvDialog({ open, appId, envs, onClose, onBusyChange, onFinished }: CopyEnvDialogProps) {
  const { t, formatNumber } = useI18n()
  const navigate = useNavigate()
  const { showToast } = useToast()
  const [step, setStep] = useState<CopyStep>('create')
  const [sourceEnv, setSourceEnv] = useState('')
  const [targetEnv, setTargetEnv] = useState('')
  const [elements, setElements] = useState<Element[]>([])
  const [selectedKeys, setSelectedKeys] = useState<Set<string>>(() => new Set())
  const [emptyStrategy, setEmptyStrategy] = useState<EmptyElementStrategy>('skip')
  const [loadingElements, setLoadingElements] = useState(false)
  const [loadError, setLoadError] = useState<{ message: string } | null>(null)
  const [copying, setCopying] = useState(false)
  const [copyError, setCopyError] = useState<{ message: string } | null>(null)
  const [progress, setProgress] = useState<CopyProgress>(() => createInitialProgress())
  const [resultStatus, setResultStatus] = useState<CopyResultStatus>('success')
  const loadSeq = useRef(0)
  const mountedRef = useRef(false)
  const openRef = useRef(open)
  const lastTargetErrorRef = useRef('')
  const lastLoadErrorRef = useRef('')
  const lastZeroCopyErrorRef = useRef('')
  const lastCopyErrorRef = useRef('')

  const loadErrorMessage = loadError ? loadError.message || t('management.copy.loadFailed') : ''
  const copyErrorMessage = copyError ? copyError.message || t('management.copy.createFailed') : ''

  const selectedElements = useMemo(() => {
    return elements.filter((element) => selectedKeys.has(getElementKey(element)))
  }, [elements, selectedKeys])

  const emptySelectedCount = useMemo(() => selectedElements.filter(isElementEmpty).length, [selectedElements])
  const skippedEstimate = emptyStrategy === 'skip' ? emptySelectedCount : 0
  const estimatedCopyCount = selectedElements.length - skippedEstimate
  const copyEmptyElements = emptyStrategy === 'copy'
  const duplicateTarget = Boolean(targetEnv && envs.includes(targetEnv))
  const sameTarget = Boolean(targetEnv && sourceEnv && targetEnv === sourceEnv)
  const targetError = duplicateTarget ? t('management.copy.exists') : sameTarget ? t('management.copy.sameEnv') : ''
  const zeroCopyError = selectedElements.length > 0 && estimatedCopyCount === 0 ? t('management.copy.zeroEstimate') : ''
  const canStart = Boolean(appId && sourceEnv && targetEnv && !targetError && !zeroCopyError && !loadingElements && estimatedCopyCount > 0 && step === 'create')
  const emptyStrategyDescription = copyEmptyElements
    ? t('management.copy.emptyOn')
    : t('management.copy.emptyOff')
  const resultCounts: Record<CopyResultStatus, number> = {
    success: progress.results.filter((result) => result.status === 'success').length,
    skipped: progress.results.filter((result) => result.status === 'skipped').length,
    failed: progress.results.filter((result) => result.status === 'failed').length,
  }
  const visibleResults = progress.results.filter((result) => result.status === resultStatus)
  const progressPercent = progress.total > 0 ? Math.round((progress.completed / progress.total) * 100) : 0
  const activeProgressStep = progress.createEnv !== 'done' ? 0 : progress.copyElements !== 'done' ? 1 : 2

  const isActive = useCallback(() => mountedRef.current && openRef.current, [])

  const loadResetState = useMemo(
    () => ({
      elements: [] as Element[],
      selectedKeys: new Set<string>(),
      loadingElements: false,
      loadError: null,
    }),
    [],
  )

  const reset = useCallback(() => {
    setStep('create')
    setSourceEnv('')
    setTargetEnv('')
    setElements(loadResetState.elements)
    setSelectedKeys(loadResetState.selectedKeys)
    setEmptyStrategy('skip')
    setLoadingElements(loadResetState.loadingElements)
    setLoadError(loadResetState.loadError)
    setCopying(false)
    setCopyError(null)
    setProgress(createInitialProgress())
    setResultStatus('success')
  }, [loadResetState])

  useEffect(() => {
    mountedRef.current = true

    return () => {
      mountedRef.current = false
      onBusyChange(false)
    }
  }, [onBusyChange])

  useEffect(() => {
    openRef.current = open
  }, [open])

  useEffect(() => {
    onBusyChange(copying)
  }, [copying, onBusyChange])

  useEffect(() => {
    if (!copying) return undefined

    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault()
      event.returnValue = t('management.copy.inProgress')
    }

    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => window.removeEventListener('beforeunload', handleBeforeUnload)
  }, [copying, t])

  useEffect(() => {
    if (!open || !appId || !sourceEnv) return undefined

    const requestId = ++loadSeq.current

    void requestAllElements(appId, sourceEnv)
      .then((items) => {
        if (!isActive() || requestId !== loadSeq.current) return
        setElements(items)
        setSelectedKeys(new Set(items.map(getElementKey)))
        setLoadError(null)
      })
      .catch((err) => {
        if (!isActive() || requestId !== loadSeq.current) return
        setElements(loadResetState.elements)
        setSelectedKeys(loadResetState.selectedKeys)
        setLoadError({ message: getErrorMessage(err, '') })
      })
      .finally(() => {
        if (isActive() && requestId === loadSeq.current) setLoadingElements(false)
      })
  }, [appId, isActive, loadResetState, open, sourceEnv])

  useEffect(() => {
    if (!targetError) {
      lastTargetErrorRef.current = ''
      return
    }
    if (lastTargetErrorRef.current === targetError) return
    lastTargetErrorRef.current = targetError
    showToast(targetError, 'warning')
  }, [showToast, targetError])

  useEffect(() => {
    if (!loadErrorMessage) {
      lastLoadErrorRef.current = ''
      return
    }
    if (lastLoadErrorRef.current === loadErrorMessage) return
    lastLoadErrorRef.current = loadErrorMessage
    showToast(loadErrorMessage, 'error')
  }, [loadErrorMessage, showToast])

  useEffect(() => {
    if (!zeroCopyError) {
      lastZeroCopyErrorRef.current = ''
      return
    }
    if (lastZeroCopyErrorRef.current === zeroCopyError) return
    lastZeroCopyErrorRef.current = zeroCopyError
    showToast(zeroCopyError, 'warning')
  }, [showToast, zeroCopyError])

  useEffect(() => {
    if (!copyErrorMessage) {
      lastCopyErrorRef.current = ''
      return
    }
    if (lastCopyErrorRef.current === copyErrorMessage) return
    lastCopyErrorRef.current = copyErrorMessage
    showToast(copyErrorMessage, 'error')
  }, [copyErrorMessage, showToast])

  function handleClose() {
    if (copying) return
    onClose()
  }

  function handleSourceEnvChange(value: string) {
    setSourceEnv(value)
    setLoadingElements(Boolean(open && appId && value))
    setLoadError(null)
  }

  function toggleElement(key: string) {
    setSelectedKeys((current) => {
      const next = new Set(current)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }

  function toggleAll(checked: boolean) {
    setSelectedKeys(checked ? new Set(elements.map(getElementKey)) : new Set())
  }

  async function copyElement(source: Element): Promise<CopyResult> {
    const key = getElementKey(source)

    try {
      const detail = await requestElement(appId, sourceEnv, key)
      if (isElementEmpty(detail) && emptyStrategy === 'skip') {
        return { key, status: 'skipped', messageKey: 'management.copy.emptySkipped' }
      }

      await createElement(appId, targetEnv, key, detail)
      return { key, status: 'success' }
    } catch (err) {
      return { key, status: 'failed', message: getErrorMessage(err, ''), messageKey: err instanceof ApiError ? undefined : 'management.copy.elementFailed' }
    }
  }

  async function runWorkers(items: Element[]) {
    let cursor = 0
    const results: CopyResult[] = []

    async function worker() {
      for (;;) {
        const index = cursor
        cursor += 1
        const item = items[index]
        if (!item) return

        const result = await copyElement(item)
        results.push(result)
        if (!isActive()) return
        setProgress((current) => ({
          ...current,
          completed: current.completed + 1,
          results: [...current.results, result],
        }))
      }
    }

    await Promise.all(Array.from({ length: Math.min(workerCount, items.length) }, () => worker()))
    return results
  }

  async function startCopy() {
    if (!canStart) return

    const items = selectedElements
    setStep('execute')
    setCopying(true)
    setCopyError(null)
    setProgress({ ...createInitialProgress(items.length), createEnv: 'doing' })

    try {
      await createEnv(appId, targetEnv)
      if (!isActive()) return
      setProgress((current) => ({ ...current, createEnv: 'done', copyElements: 'doing' }))
      const results = await runWorkers(items)
      if (!isActive()) return
      setResultStatus(results.some((result) => result.status === 'failed') ? 'failed' : results.some((result) => result.status === 'skipped') ? 'skipped' : 'success')
      setProgress((current) => ({ ...current, copyElements: 'done', complete: 'done' }))
    } catch (err) {
      if (!isActive()) return
      setCopyError({ message: getErrorMessage(err, '') })
      setProgress((current) => ({ ...current, createEnv: 'failed', copyElements: 'pending', complete: 'pending' }))
      setCopying(false)
      return
    }

    try {
      await onFinished()
    } catch {
      // Ignore refresh errors here so a completed copy stays completed.
    } finally {
      if (mountedRef.current) setCopying(false)
    }
  }

  function viewCopiedEnv() {
    onClose()
    navigate(`/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(targetEnv)}/elements`)
  }

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      onTransitionExited={reset}
      fullWidth
      maxWidth="md"
      aria-labelledby="copy-environment-dialog-title"
    >
      <DialogTitle id="copy-environment-dialog-title">{t('management.copy.title')}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ mt: 1 }}>
          <Stack direction="row" spacing={1} alignItems="center">
            <Typography variant="overline" color={step === 'create' ? 'primary' : 'text.secondary'}>
              {t('management.copy.taskCreation')}
            </Typography>
            <Divider flexItem orientation="vertical" />
            <Typography variant="overline" color={step === 'execute' ? 'primary' : 'text.secondary'}>
              {t('management.copy.taskExecution')}
            </Typography>
          </Stack>

          {step === 'create' ? (
            <Stack spacing={2}>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                <FormControl fullWidth disabled={!appId || loadingElements} required>
                  <InputLabel id="copy-source-env-label" required>{t('management.copy.sourceEnv')}</InputLabel>
                  <Select
                    labelId="copy-source-env-label"
                    label={t('management.copy.sourceEnvRequired')}
                    value={sourceEnv}
                    onChange={(event) => handleSourceEnvChange(String(event.target.value))}
                  >
                    {envs.map((env) => (
                      <MenuItem key={env} value={env}>
                        {env}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>

                <TextField
                  label={t('management.copy.toEnv')}
                  value={targetEnv}
                  onChange={(event) => setTargetEnv(event.target.value.trim().toLowerCase())}
                  error={Boolean(targetError)}
                  helperText={targetError || t('management.copy.targetHelp')}
                  fullWidth
                  disabled={!appId}
                  required
                />
              </Stack>

              <FormControl>
                <Stack direction="row" spacing={1.5} alignItems="center" flexWrap="wrap">
                  <Typography color={!copyEmptyElements ? 'warning.dark' : 'text.secondary'} sx={{ fontWeight: !copyEmptyElements ? 700 : 500 }}>
                    {t('management.copy.skipEmpty')}
                  </Typography>
                  <Switch
                    checked={copyEmptyElements}
                    onChange={(event) => setEmptyStrategy(event.target.checked ? 'copy' : 'skip')}
                    slotProps={{ input: { 'aria-label': t('management.copy.copyEmptyElements'), 'aria-describedby': 'copy-empty-elements-help' } }}
                  />
                  <Typography color={copyEmptyElements ? 'primary.main' : 'text.secondary'} sx={{ fontWeight: copyEmptyElements ? 700 : 500 }}>
                    {t('management.copy.copyEmpty')}
                  </Typography>
                </Stack>
                <FormHelperText id="copy-empty-elements-help">{emptyStrategyDescription}</FormHelperText>
              </FormControl>

              {loadingElements && <LinearProgress aria-label={t('management.copy.loadingElements')} />}

              {sourceEnv && !loadingElements && !loadError && (
                <Paper variant="outlined">
                  <Stack spacing={1} sx={{ p: 2 }}>
                    <FormControlLabel
                      control={
                        <Checkbox
                          checked={elements.length > 0 && selectedKeys.size === elements.length}
                          indeterminate={selectedKeys.size > 0 && selectedKeys.size < elements.length}
                          onChange={(event) => toggleAll(event.target.checked)}
                        />
                      }
                      label={t('management.copy.selectAll', { selected: formatNumber(selectedKeys.size), total: formatNumber(elements.length) })}
                    />
                    <Divider />
                    {elements.length === 0 ? (
                      <Typography color="text.secondary">{t('management.copy.emptySource')}</Typography>
                    ) : (
                      <Box role="region" aria-label={t('management.copy.elementsList')} sx={{ maxHeight: '240px', overflowY: 'auto' }}>
                        <List dense disablePadding>
                          {elements.map((element) => {
                            const key = getElementKey(element)
                            const empty = isElementEmpty(element)
                            const checked = selectedKeys.has(key)
                            return (
                              <ListItem key={key} disablePadding>
                                <ListItemButton onClick={() => toggleElement(key)}>
                                  <ListItemIcon>
                                    <Checkbox
                                      edge="start"
                                      checked={checked}
                                      tabIndex={-1}
                                      inputProps={{ 'aria-label': key }}
                                      onClick={(event) => event.stopPropagation()}
                                      onChange={() => toggleElement(key)}
                                    />
                                  </ListItemIcon>
                                  <ListItemText
                                    primary={key}
                                    secondary={t(empty ? 'management.copy.elementSummaryEmpty' : 'management.copy.elementSummary', { version: element.metadata.usingVersion || '-', type: getContentTypeLabel(element.metadata.contentType) })}
                                  />
                                </ListItemButton>
                              </ListItem>
                            )
                          })}
                        </List>
                      </Box>
                    )}
                  </Stack>
                </Paper>
              )}

              <Paper variant="outlined" sx={{ p: 2 }}>
                <Stack spacing={1.5}>
                  <Typography variant="subtitle2">{t('management.copy.summary')}</Typography>
                  <Box data-testid="copy-summary-grid" sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', columnGap: 3, rowGap: 1 }}>
                    <Box data-testid="copy-summary-source-env" sx={{ display: 'grid', gridTemplateColumns: '150px minmax(0, 1fr)', alignItems: 'baseline', columnGap: 1 }}>
                      <Typography data-testid="copy-summary-source-env-label" sx={{ fontWeight: 700 }} color="text.secondary">{t('management.copy.sourceEnv')}</Typography>
                      <Typography data-testid="copy-summary-source-env-value" sx={{ overflowWrap: 'anywhere' }}>{sourceEnv || '-'}</Typography>
                    </Box>
                    <Box data-testid="copy-summary-target-env" sx={{ display: 'grid', gridTemplateColumns: '150px minmax(0, 1fr)', alignItems: 'baseline', columnGap: 1 }}>
                      <Typography data-testid="copy-summary-target-env-label" sx={{ fontWeight: 700 }} color="text.secondary">{t('management.copy.targetEnv')}</Typography>
                      <Typography data-testid="copy-summary-target-env-value" sx={{ overflowWrap: 'anywhere' }}>{targetEnv || '-'}</Typography>
                    </Box>
                    <Box data-testid="copy-summary-selected-elements" sx={{ display: 'grid', gridTemplateColumns: '150px minmax(0, 1fr)', alignItems: 'baseline', columnGap: 1 }}>
                      <Typography data-testid="copy-summary-selected-elements-label" sx={{ fontWeight: 700 }} color="text.secondary">{t('management.copy.selectedElements')}</Typography>
                      <Typography data-testid="copy-summary-selected-elements-value">{formatNumber(selectedElements.length)}</Typography>
                    </Box>
                    <Box data-testid="copy-summary-empty-elements" sx={{ display: 'grid', gridTemplateColumns: '150px minmax(0, 1fr)', alignItems: 'baseline', columnGap: 1 }}>
                      <Typography data-testid="copy-summary-empty-elements-label" sx={{ fontWeight: 700 }} color="text.secondary">{t('management.copy.emptyElements')}</Typography>
                      <Typography data-testid="copy-summary-empty-elements-value">{formatNumber(emptySelectedCount)}</Typography>
                    </Box>
                    <Box data-testid="copy-summary-estimated-skipped" sx={{ display: 'grid', gridTemplateColumns: '150px minmax(0, 1fr)', alignItems: 'baseline', columnGap: 1 }}>
                      <Typography data-testid="copy-summary-estimated-skipped-label" sx={{ fontWeight: 700 }} color="text.secondary">{t('management.copy.estimatedSkipped')}</Typography>
                      <Typography data-testid="copy-summary-estimated-skipped-value">{formatNumber(skippedEstimate)}</Typography>
                    </Box>
                    <Box data-testid="copy-summary-estimated-copy" sx={{ display: 'grid', gridTemplateColumns: '150px minmax(0, 1fr)', alignItems: 'baseline', columnGap: 1 }}>
                      <Typography data-testid="copy-summary-estimated-copy-label" sx={{ fontWeight: 700 }} color="text.secondary">{t('management.copy.estimatedCopy')}</Typography>
                      <Typography data-testid="copy-summary-estimated-copy-value" color={estimatedCopyCount === 0 ? 'error.main' : 'text.primary'}>{formatNumber(estimatedCopyCount)}</Typography>
                    </Box>
                  </Box>
                </Stack>
              </Paper>
            </Stack>
          ) : (
            <Stack spacing={2}>
              <Paper variant="outlined" sx={{ p: 2 }}>
                <Stack spacing={2}>
                  <Typography variant="subtitle2">{t('management.copy.progress')}</Typography>
                  <Stepper activeStep={activeProgressStep} alternativeLabel>
                    <Step completed={progress.createEnv === 'done'}>
                      <StepLabel error={progress.createEnv === 'failed'} optional={<Typography variant="caption">{t(`management.copy.progress.${progress.createEnv}`)}</Typography>}>
                        {t('management.copy.createEnv')}
                      </StepLabel>
                    </Step>
                    <Step completed={progress.copyElements === 'done'}>
                      <StepLabel error={progress.copyElements === 'failed'} optional={<Typography variant="caption">{t(`management.copy.progress.${progress.copyElements}`)}</Typography>}>
                        {t('management.copy.copyElements')}
                      </StepLabel>
                    </Step>
                    <Step completed={progress.complete === 'done'}>
                      <StepLabel error={progress.complete === 'failed'} optional={<Typography variant="caption">{t(`management.copy.progress.${progress.complete}`)}</Typography>}>
                        {t('management.copy.complete')}
                      </StepLabel>
                    </Step>
                  </Stepper>
                  <Stack spacing={0.75}>
                    <LinearProgress
                      variant="determinate"
                      value={progressPercent}
                      aria-label={t('management.copy.elementsProgress')}
                    />
                    <Typography variant="body2" color="text.secondary">
                      {t('management.copy.processed', { completed: formatNumber(progress.completed), total: formatNumber(progress.total) })}
                    </Typography>
                  </Stack>
                </Stack>
              </Paper>

              {progress.results.length > 0 && (
                <Paper variant="outlined" sx={{ p: 2 }} role="region" aria-label={t('management.copy.results')}>
                  <Stack spacing={1.5}>
                    <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.5} alignItems={{ xs: 'stretch', md: 'center' }} justifyContent="space-between">
                      <Typography variant="subtitle2">{t('management.copy.resultsTitle')}</Typography>
                      <Tabs value={resultStatus} onChange={(_, value: CopyResultStatus) => setResultStatus(value)} aria-label={t('management.copy.resultStates')}>
                        {copyResultStatuses.map((status) => (
                          <Tab
                            key={status}
                            value={status}
                            label={<CopyResultCount status={status} count={resultCounts[status]} testId={`copy-results-${status}-count`} />}
                            sx={{ minHeight: 40, textTransform: 'none' }}
                          />
                        ))}
                      </Tabs>
                    </Stack>
                    <TableContainer sx={{ maxHeight: 260, overflowY: 'auto' }}>
                      <Table stickyHeader size="small" aria-label={t('management.copy.results')}>
                        <TableHead>
                          <TableRow>
                            <TableCell>{t('management.copy.key')}</TableCell>
                            <TableCell>{t('management.copy.state')}</TableCell>
                            <TableCell>{t('management.copy.reason')}</TableCell>
                          </TableRow>
                        </TableHead>
                        <TableBody>
                          {visibleResults.map((result) => {
                            const meta = copyResultStatusMeta[result.status]

                            return (
                              <TableRow key={`${result.key}-${result.status}`} sx={{ backgroundColor: meta.backgroundColor }}>
                                <TableCell sx={{ color: meta.color, overflowWrap: 'anywhere' }}>{result.key}</TableCell>
                                <TableCell sx={{ color: meta.color }}><CopyResultState status={result.status} /></TableCell>
                                <TableCell sx={{ color: meta.color, overflowWrap: 'anywhere' }}>{result.messageKey ? t(result.messageKey) : result.message || '-'}</TableCell>
                              </TableRow>
                            )
                          })}
                        </TableBody>
                      </Table>
                    </TableContainer>
                  </Stack>
                </Paper>
              )}
            </Stack>
          )}
        </Stack>
      </DialogContent>
      <DialogActions>
        {step === 'create' ? (
          <>
            <Button onClick={handleClose} disabled={copying}>
              {t('management.common.cancel')}
            </Button>
            <Button variant="contained" onClick={() => void startCopy()} disabled={!canStart}>
              {t('management.copy.start')}
            </Button>
          </>
        ) : (
          <>
            <Button onClick={handleClose} disabled={copying}>
              {t('management.common.close')}
            </Button>
            <Button variant="contained" onClick={viewCopiedEnv} disabled={copying || progress.createEnv !== 'done'}>
              {t('management.copy.viewEnv')}
            </Button>
          </>
        )}
      </DialogActions>
    </Dialog>
  )
}
