import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline'
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline'
import WarningAmberIcon from '@mui/icons-material/WarningAmber'
import {
  Alert,
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
import { contentTypes, type Element, type ElementsResponse } from '../../domain/types'
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

async function createEnv(appId: string, env: string) {
  return apiRequest<void>(`/api/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}`, { method: 'POST' })
}

async function createElement(appId: string, env: string, key: string, element: Element) {
  return apiRequest<void>(
    `/api/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}`,
    jsonBody({ raw: decodeRaw(element.raw || ''), contentType: element.metadata.contentType || contentTypes[0].value }),
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

const copyResultStatusMeta: Record<CopyResultStatus, { label: string; color: string; backgroundColor: string }> = {
  success: { label: 'Success', color: 'rgb(46, 125, 50)', backgroundColor: 'rgba(46, 125, 50, 0.08)' },
  skipped: { label: 'Skipped', color: 'rgb(97, 97, 97)', backgroundColor: 'rgba(97, 97, 97, 0.08)' },
  failed: { label: 'Failed', color: 'rgb(211, 47, 47)', backgroundColor: 'rgba(211, 47, 47, 0.08)' },
}

function getCopyResultIcon(status: CopyResultStatus) {
  if (status === 'success') return <CheckCircleOutlineIcon fontSize="small" aria-hidden="true" />
  if (status === 'skipped') return <WarningAmberIcon fontSize="small" aria-hidden="true" />
  return <ErrorOutlineIcon fontSize="small" aria-hidden="true" />
}

function CopyResultCount({ status, count, testId }: { status: CopyResultStatus; count: number; testId: string }) {
  const meta = copyResultStatusMeta[status]

  return (
    <Stack component="span" data-testid={testId} direction="row" spacing={0.5} alignItems="center" sx={{ color: meta.color, fontWeight: 700 }}>
      {getCopyResultIcon(status)}
      <Typography component="span" variant="body2" sx={{ color: 'inherit', fontWeight: 700 }}>
        {meta.label} {count}
      </Typography>
    </Stack>
  )
}

function CopyResultState({ status }: { status: CopyResultStatus }) {
  const meta = copyResultStatusMeta[status]

  return (
    <Stack component="span" direction="row" spacing={0.5} alignItems="center" sx={{ color: meta.color, fontWeight: 700 }}>
      {getCopyResultIcon(status)}
      <Typography component="span" sx={{ color: 'inherit', fontSize: 'inherit', fontWeight: 700, lineHeight: 'inherit' }}>
        {meta.label}
      </Typography>
    </Stack>
  )
}

export function CopyEnvDialog({ open, appId, envs, onClose, onBusyChange, onFinished }: CopyEnvDialogProps) {
  const navigate = useNavigate()
  const [step, setStep] = useState<CopyStep>('create')
  const [sourceEnv, setSourceEnv] = useState('')
  const [targetEnv, setTargetEnv] = useState('')
  const [elements, setElements] = useState<Element[]>([])
  const [selectedKeys, setSelectedKeys] = useState<Set<string>>(() => new Set())
  const [emptyStrategy, setEmptyStrategy] = useState<EmptyElementStrategy>('skip')
  const [loadingElements, setLoadingElements] = useState(false)
  const [loadError, setLoadError] = useState('')
  const [copying, setCopying] = useState(false)
  const [copyError, setCopyError] = useState('')
  const [progress, setProgress] = useState<CopyProgress>(() => createInitialProgress())
  const [resultStatus, setResultStatus] = useState<CopyResultStatus>('success')
  const loadSeq = useRef(0)
  const mountedRef = useRef(false)
  const openRef = useRef(open)

  const selectedElements = useMemo(() => {
    return elements.filter((element) => selectedKeys.has(getElementKey(element)))
  }, [elements, selectedKeys])

  const emptySelectedCount = useMemo(() => selectedElements.filter(isElementEmpty).length, [selectedElements])
  const skippedEstimate = emptyStrategy === 'skip' ? emptySelectedCount : 0
  const estimatedCopyCount = selectedElements.length - skippedEstimate
  const copyEmptyElements = emptyStrategy === 'copy'
  const duplicateTarget = Boolean(targetEnv && envs.includes(targetEnv))
  const sameTarget = Boolean(targetEnv && sourceEnv && targetEnv === sourceEnv)
  const targetError = duplicateTarget ? 'Environment already exists' : sameTarget ? 'Target env must be different from source env' : ''
  const zeroCopyError = selectedElements.length > 0 && estimatedCopyCount === 0 ? 'Estimated copy is 0. Enable Copy empty elements or select non-empty elements.' : ''
  const canStart = Boolean(appId && sourceEnv && targetEnv && !targetError && !zeroCopyError && !loadingElements && estimatedCopyCount > 0 && step === 'create')
  const emptyStrategyDescription = copyEmptyElements
    ? 'Copy empty elements is on. Empty elements will be copied and may fail server validation.'
    : 'Copy empty elements is off. Empty elements will be skipped.'
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
      loadError: '',
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
    setCopyError('')
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
      event.returnValue = 'copy in progress'
    }

    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => window.removeEventListener('beforeunload', handleBeforeUnload)
  }, [copying])

  useEffect(() => {
    if (!open || !appId || !sourceEnv) return undefined

    const requestId = ++loadSeq.current

    void requestAllElements(appId, sourceEnv)
      .then((items) => {
        if (!isActive() || requestId !== loadSeq.current) return
        setElements(items)
        setSelectedKeys(new Set(items.map(getElementKey)))
        setLoadError('')
      })
      .catch((err) => {
        if (!isActive() || requestId !== loadSeq.current) return
        setElements(loadResetState.elements)
        setSelectedKeys(loadResetState.selectedKeys)
        setLoadError(getErrorMessage(err, 'failed to load source elements'))
      })
      .finally(() => {
        if (isActive() && requestId === loadSeq.current) setLoadingElements(false)
      })
  }, [appId, isActive, loadResetState, open, sourceEnv])

  function handleClose() {
    if (copying) return
    onClose()
  }

  function handleSourceEnvChange(value: string) {
    setSourceEnv(value)
    setLoadingElements(Boolean(open && appId && value))
    setLoadError('')
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
        return { key, status: 'skipped', message: 'empty element skipped' }
      }

      await createElement(appId, targetEnv, key, detail)
      return { key, status: 'success' }
    } catch (err) {
      return { key, status: 'failed', message: getErrorMessage(err, 'failed to copy element') }
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
    setCopyError('')
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
      setCopyError(getErrorMessage(err, 'failed to create target environment'))
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
      <DialogTitle id="copy-environment-dialog-title">Copy environment</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ mt: 1 }}>
          <Stack direction="row" spacing={1} alignItems="center">
            <Typography variant="overline" color={step === 'create' ? 'primary' : 'text.secondary'}>
              Task creation
            </Typography>
            <Divider flexItem orientation="vertical" />
            <Typography variant="overline" color={step === 'execute' ? 'primary' : 'text.secondary'}>
              Task execution
            </Typography>
          </Stack>

          {step === 'create' ? (
            <Stack spacing={2}>
              <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
                <FormControl fullWidth disabled={!appId || loadingElements} required>
                  <InputLabel id="copy-source-env-label" required>Source env</InputLabel>
                  <Select
                    labelId="copy-source-env-label"
                    label="Source env *"
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
                  label="To env"
                  value={targetEnv}
                  onChange={(event) => setTargetEnv(event.target.value.trim().toLowerCase())}
                  error={Boolean(targetError)}
                  helperText={targetError || 'New environment name. Values are saved lowercase.'}
                  fullWidth
                  disabled={!appId}
                  required
                />
              </Stack>

              <FormControl>
                <Stack direction="row" spacing={1.5} alignItems="center" flexWrap="wrap">
                  <Typography color={!copyEmptyElements ? 'warning.dark' : 'text.secondary'} sx={{ fontWeight: !copyEmptyElements ? 700 : 500 }}>
                    Skip empty
                  </Typography>
                  <Switch
                    checked={copyEmptyElements}
                    onChange={(event) => setEmptyStrategy(event.target.checked ? 'copy' : 'skip')}
                    slotProps={{ input: { 'aria-label': 'Copy empty elements', 'aria-describedby': 'copy-empty-elements-help' } }}
                  />
                  <Typography color={copyEmptyElements ? 'primary.main' : 'text.secondary'} sx={{ fontWeight: copyEmptyElements ? 700 : 500 }}>
                    Copy empty
                  </Typography>
                </Stack>
                <FormHelperText id="copy-empty-elements-help">{emptyStrategyDescription}</FormHelperText>
              </FormControl>

              {loadingElements && <LinearProgress aria-label="Loading source elements" />}
              {loadError && <Alert severity="error">{loadError}</Alert>}

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
                      label={`Select all elements (${selectedKeys.size}/${elements.length})`}
                    />
                    <Divider />
                    {elements.length === 0 ? (
                      <Typography color="text.secondary">No elements found in source environment.</Typography>
                    ) : (
                      <Box role="region" aria-label="Copy elements list" sx={{ maxHeight: '240px', overflowY: 'auto' }}>
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
                                    secondary={`using v${element.metadata.usingVersion || '-'} · ${getContentTypeLabel(element.metadata.contentType)}${empty ? ' · empty' : ''}`}
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
                  <Typography variant="subtitle2">Copy summary</Typography>
                  <Box data-testid="copy-summary-grid" sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', columnGap: 3, rowGap: 1 }}>
                    <Box data-testid="copy-summary-source-env" sx={{ display: 'grid', gridTemplateColumns: '150px minmax(0, 1fr)', alignItems: 'baseline', columnGap: 1 }}>
                      <Typography data-testid="copy-summary-source-env-label" sx={{ fontWeight: 700 }} color="text.secondary">Source env</Typography>
                      <Typography data-testid="copy-summary-source-env-value" sx={{ overflowWrap: 'anywhere' }}>{sourceEnv || '-'}</Typography>
                    </Box>
                    <Box data-testid="copy-summary-target-env" sx={{ display: 'grid', gridTemplateColumns: '150px minmax(0, 1fr)', alignItems: 'baseline', columnGap: 1 }}>
                      <Typography data-testid="copy-summary-target-env-label" sx={{ fontWeight: 700 }} color="text.secondary">Target env</Typography>
                      <Typography data-testid="copy-summary-target-env-value" sx={{ overflowWrap: 'anywhere' }}>{targetEnv || '-'}</Typography>
                    </Box>
                    <Box data-testid="copy-summary-selected-elements" sx={{ display: 'grid', gridTemplateColumns: '150px minmax(0, 1fr)', alignItems: 'baseline', columnGap: 1 }}>
                      <Typography data-testid="copy-summary-selected-elements-label" sx={{ fontWeight: 700 }} color="text.secondary">Selected elements</Typography>
                      <Typography data-testid="copy-summary-selected-elements-value">{selectedElements.length}</Typography>
                    </Box>
                    <Box data-testid="copy-summary-empty-elements" sx={{ display: 'grid', gridTemplateColumns: '150px minmax(0, 1fr)', alignItems: 'baseline', columnGap: 1 }}>
                      <Typography data-testid="copy-summary-empty-elements-label" sx={{ fontWeight: 700 }} color="text.secondary">Empty elements</Typography>
                      <Typography data-testid="copy-summary-empty-elements-value">{emptySelectedCount}</Typography>
                    </Box>
                    <Box data-testid="copy-summary-estimated-skipped" sx={{ display: 'grid', gridTemplateColumns: '150px minmax(0, 1fr)', alignItems: 'baseline', columnGap: 1 }}>
                      <Typography data-testid="copy-summary-estimated-skipped-label" sx={{ fontWeight: 700 }} color="text.secondary">Estimated skipped</Typography>
                      <Typography data-testid="copy-summary-estimated-skipped-value">{skippedEstimate}</Typography>
                    </Box>
                    <Box data-testid="copy-summary-estimated-copy" sx={{ display: 'grid', gridTemplateColumns: '150px minmax(0, 1fr)', alignItems: 'baseline', columnGap: 1 }}>
                      <Typography data-testid="copy-summary-estimated-copy-label" sx={{ fontWeight: 700 }} color="text.secondary">Estimated copy</Typography>
                      <Typography data-testid="copy-summary-estimated-copy-value" color={estimatedCopyCount === 0 ? 'error.main' : 'text.primary'}>{estimatedCopyCount}</Typography>
                    </Box>
                  </Box>
                  {zeroCopyError && <Alert severity="warning">{zeroCopyError}</Alert>}
                </Stack>
              </Paper>
            </Stack>
          ) : (
            <Stack spacing={2}>
              {copyError && <Alert severity="error">{copyError}</Alert>}
              <Paper variant="outlined" sx={{ p: 2 }}>
                <Stack spacing={2}>
                  <Typography variant="subtitle2">Progress</Typography>
                  <Stepper activeStep={activeProgressStep} alternativeLabel>
                    <Step completed={progress.createEnv === 'done'}>
                      <StepLabel error={progress.createEnv === 'failed'} optional={<Typography variant="caption">{progress.createEnv}</Typography>}>
                        Create env
                      </StepLabel>
                    </Step>
                    <Step completed={progress.copyElements === 'done'}>
                      <StepLabel error={progress.copyElements === 'failed'} optional={<Typography variant="caption">{progress.copyElements}</Typography>}>
                        Copy elements
                      </StepLabel>
                    </Step>
                    <Step completed={progress.complete === 'done'}>
                      <StepLabel error={progress.complete === 'failed'} optional={<Typography variant="caption">{progress.complete}</Typography>}>
                        Complete
                      </StepLabel>
                    </Step>
                  </Stepper>
                  <Stack spacing={0.75}>
                    <LinearProgress
                      variant="determinate"
                      value={progressPercent}
                      aria-label="Copy elements progress"
                    />
                    <Typography variant="body2" color="text.secondary">
                      {progress.completed}/{progress.total} elements processed
                    </Typography>
                  </Stack>
                </Stack>
              </Paper>

              {progress.results.length > 0 && (
                <Paper variant="outlined" sx={{ p: 2 }} role="region" aria-label="Copy results">
                  <Stack spacing={1.5}>
                    <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.5} alignItems={{ xs: 'stretch', md: 'center' }} justifyContent="space-between">
                      <Typography variant="subtitle2">Results</Typography>
                      <Tabs value={resultStatus} onChange={(_, value: CopyResultStatus) => setResultStatus(value)} aria-label="Copy result states">
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
                      <Table stickyHeader size="small" aria-label="Copy results">
                        <TableHead>
                          <TableRow>
                            <TableCell>Key</TableCell>
                            <TableCell>State</TableCell>
                            <TableCell>Reason</TableCell>
                          </TableRow>
                        </TableHead>
                        <TableBody>
                          {visibleResults.map((result) => {
                            const meta = copyResultStatusMeta[result.status]

                            return (
                              <TableRow key={`${result.key}-${result.status}`} sx={{ backgroundColor: meta.backgroundColor }}>
                                <TableCell sx={{ color: meta.color, overflowWrap: 'anywhere' }}>{result.key}</TableCell>
                                <TableCell sx={{ color: meta.color }}><CopyResultState status={result.status} /></TableCell>
                                <TableCell sx={{ color: meta.color, overflowWrap: 'anywhere' }}>{result.message || '-'}</TableCell>
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
              Cancel
            </Button>
            <Button variant="contained" onClick={() => void startCopy()} disabled={!canStart}>
              Start copy
            </Button>
          </>
        ) : (
          <>
            <Button onClick={handleClose} disabled={copying}>
              Close
            </Button>
            <Button variant="contained" onClick={viewCopiedEnv} disabled={copying || progress.createEnv !== 'done'}>
              View copied env
            </Button>
          </>
        )}
      </DialogActions>
    </Dialog>
  )
}
