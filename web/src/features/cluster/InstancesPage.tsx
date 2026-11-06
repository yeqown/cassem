import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import {
  Autocomplete,
  Box,
  Button,
  Link,
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
import { Link as RouterLink, useSearchParams } from 'react-router-dom'
import { EmptyState, ErrorState, LoadingState } from '../../components/StateView'
import { useErrorState } from '../../components/useErrorState'
import type { AppsResponse, ElementsResponse, EnvsResponse, Instance, InstancesResponse } from '../../domain/types'
import { useI18n, type Translator } from '../../i18n'
import { isOperationsMessageKey } from '../../i18n/catalogs/operations'
import { ApiError, apiRequest, buildQuery } from '../../lib/api'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback
}

function uniqueStrings(values: string[]) {
  return Array.from(new Set(values.filter(Boolean)))
}

function renderElementLink(label: string, to: string) {
  if (!to) return label

  return (
    <Link component={RouterLink} to={to} underline="hover" color="primary" sx={{ fontWeight: 600, overflowWrap: 'anywhere' }}>
      {label}
    </Link>
  )
}

function renderTargets(instance: Instance) {
  const targets = instance.targets || []
  if (targets.length === 0) return '-'

  return (
    <Stack spacing={0.75}>
      {targets.map((target, index) => {
        const app = target.app || ''
        const env = target.env || ''
        const key = target.key || ''
        const rowKey = `${app}/${env}/${key}/${index}`

        return (
          <Stack key={rowKey} direction="row" spacing={0.75} alignItems="center" flexWrap="wrap">
            {renderElementLink(app, app ? `/apps/${encodeURIComponent(app)}/envs` : '')}
            <Typography component="span" color="text.secondary">/</Typography>
            {renderElementLink(env, app && env ? `/apps/${encodeURIComponent(app)}/envs/${encodeURIComponent(env)}/elements` : '')}
            <Typography component="span" color="text.secondary">/</Typography>
            {renderElementLink(key, app && env && key ? `/apps/${encodeURIComponent(app)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}` : '')}
          </Stack>
        )
      })}
    </Stack>
  )
}

function formatLastSeen(timestamp: number | undefined, t: Translator) {
  if (!timestamp) return '-'

  const elapsed = Math.max(0, Math.floor(Date.now() / 1000) - timestamp)
  const hours = Math.floor(elapsed / 3600)
  const minutes = Math.floor((elapsed % 3600) / 60)
  const seconds = elapsed % 60

  if (hours > 0) return t('operations.instances.hoursAgo', { hours, minutes })
  if (minutes > 0) return t('operations.instances.minutesAgo', { minutes, seconds })
  return t('operations.instances.secondsAgo', { seconds })
}

async function requestApps() {
  return apiRequest<AppsResponse>(`/api/apps${buildQuery({ limit: 100 })}`)
}

async function requestEnvs(app: string) {
  return apiRequest<EnvsResponse>(`/api/apps/${encodeURIComponent(app)}/envs${buildQuery({ limit: 100 })}`)
}

async function requestElements(app: string, env: string) {
  return apiRequest<ElementsResponse>(`/api/apps/${encodeURIComponent(app)}/envs/${encodeURIComponent(env)}/elements${buildQuery({ limit: 100 })}`)
}

async function requestInstances() {
  return apiRequest<InstancesResponse>(`/api/cluster/instances${buildQuery({ limit: 100 })}`)
}

async function requestFilteredInstances(app: string, env: string, key: string) {
  return apiRequest<InstancesResponse>(`/api/cluster/instances/filter${buildQuery({ app, env, key })}`)
}

type InstancesPageFlowProps = {
  initialApp: string
  initialEnv: string
  initialKey: string
}

function InstancesPageFlow({ initialApp, initialEnv, initialKey }: InstancesPageFlowProps) {
  const { t, formatDateTime } = useI18n()
  const [app, setApp] = useState(initialApp)
  const [env, setEnv] = useState(initialEnv)
  const [key, setKey] = useState(initialKey)
  const [appOptions, setAppOptions] = useState<string[]>([])
  const [envOptions, setEnvOptions] = useState<string[]>([])
  const [keyOptions, setKeyOptions] = useState<string[]>([])
  const [candidateLoading, setCandidateLoading] = useState(false)
  const [instances, setInstances] = useState<Instance[]>([])
  const [detail, setDetail] = useState<unknown>(null)
  const [loading, setLoading] = useState(true)
  const [detailLoading, setDetailLoading] = useState(false)
  const [error, setError] = useErrorState()
  const [detailError, setDetailError] = useState('')
  const [filterError, setFilterError] = useState('')
  const requestSeq = useRef(0)
  const detailRequestSeq = useRef(0)
  const candidateSeq = useRef(0)
  const mountedRef = useRef(false)
  const lastLoadKeyRef = useRef('')

  const loadAll = useCallback(async () => {
    const requestId = ++requestSeq.current

    detailRequestSeq.current += 1
    setDetail(null)
    setDetailError('')
    setDetailLoading(false)

    try {
      const data = await requestInstances()
      if (!mountedRef.current || requestId !== requestSeq.current) return
      setInstances(data.instances || [])
      setError('')
    } catch (err) {
      if (!mountedRef.current || requestId !== requestSeq.current) return
      setInstances([])
      setError(getErrorMessage(err, 'operations.instances.loadFailed'))
    } finally {
      if (mountedRef.current && requestId === requestSeq.current) setLoading(false)
    }
  }, [setError])

  const applyFilter = useCallback(
    async (nextApp: string, nextEnv: string, nextKey: string) => {
      if (!nextApp || !nextEnv || !nextKey) {
        await loadAll()
        return
      }

      const requestId = ++requestSeq.current

      detailRequestSeq.current += 1
      setDetail(null)
      setDetailError('')
      setDetailLoading(false)

      try {
        const data = await requestFilteredInstances(nextApp, nextEnv, nextKey)
        if (!mountedRef.current || requestId !== requestSeq.current) return
        setInstances(data.instances || [])
        setError('')
      } catch (err) {
        if (!mountedRef.current || requestId !== requestSeq.current) return
        setInstances([])
        setError(getErrorMessage(err, 'operations.instances.filterFailed'))
      } finally {
        if (mountedRef.current && requestId === requestSeq.current) setLoading(false)
      }
    },
    [loadAll, setError],
  )

  const loadApps = useCallback(async () => {
    const requestId = ++candidateSeq.current
    setCandidateLoading(true)

    try {
      const data = await requestApps()
      if (!mountedRef.current || requestId !== candidateSeq.current) return
      setAppOptions(uniqueStrings((data.apps || []).map((item) => item.id)))
      setFilterError('')
    } catch (err) {
      if (!mountedRef.current || requestId !== candidateSeq.current) return
      setAppOptions([])
      setFilterError(getErrorMessage(err, 'operations.instances.appsFailed'))
    } finally {
      if (mountedRef.current && requestId === candidateSeq.current) setCandidateLoading(false)
    }
  }, [])

  async function loadEnvCandidates(nextApp: string) {
    const requestId = ++candidateSeq.current
    setCandidateLoading(true)

    if (!nextApp) {
      setEnvOptions([])
      setKeyOptions([])
      setCandidateLoading(false)
      return
    }

    try {
      const data = await requestEnvs(nextApp)
      if (!mountedRef.current || requestId !== candidateSeq.current) return
      setEnvOptions(data.envs || [])
      setKeyOptions([])
      setFilterError('')
    } catch (err) {
      if (!mountedRef.current || requestId !== candidateSeq.current) return
      setEnvOptions([])
      setKeyOptions([])
      setFilterError(getErrorMessage(err, 'operations.instances.envsFailed'))
    } finally {
      if (mountedRef.current && requestId === candidateSeq.current) setCandidateLoading(false)
    }
  }

  async function loadKeyCandidates(nextApp: string, nextEnv: string) {
    const requestId = ++candidateSeq.current
    setCandidateLoading(true)

    if (!nextApp || !nextEnv) {
      setKeyOptions([])
      setCandidateLoading(false)
      return
    }

    try {
      const data = await requestElements(nextApp, nextEnv)
      if (!mountedRef.current || requestId !== candidateSeq.current) return
      setKeyOptions(uniqueStrings((data.elements || []).map((element) => element.metadata?.key || '')))
      setFilterError('')
    } catch (err) {
      if (!mountedRef.current || requestId !== candidateSeq.current) return
      setKeyOptions([])
      setFilterError(getErrorMessage(err, 'operations.instances.keysFailed'))
    } finally {
      if (mountedRef.current && requestId === candidateSeq.current) setCandidateLoading(false)
    }
  }

  useEffect(() => {
    mountedRef.current = true

    const loadKey = JSON.stringify({ initialApp, initialEnv, initialKey })
    if (lastLoadKeyRef.current !== loadKey) {
      lastLoadKeyRef.current = loadKey
      queueMicrotask(() => {
        void loadApps()
        if (initialApp) void loadEnvCandidates(initialApp)
        if (initialApp && initialEnv) void loadKeyCandidates(initialApp, initialEnv)
        if (initialApp && initialEnv && initialKey) {
          void applyFilter(initialApp, initialEnv, initialKey)
        } else {
          void loadAll()
        }
      })
    }

    return () => {
      mountedRef.current = false
    }
  }, [applyFilter, initialApp, initialEnv, initialKey, loadAll, loadApps])

  async function handleFilter(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const nextApp = app.trim()
    const nextEnv = env.trim()
    const nextKey = key.trim()

    setLoading(true)
    setError('')
    setFilterError('')
    await applyFilter(nextApp, nextEnv, nextKey)
  }

  async function handleRefreshAll() {
    setLoading(true)
    setError('')
    setFilterError('')
    setApp('')
    setEnv('')
    setKey('')
    setEnvOptions([])
    setKeyOptions([])
    await loadAll()
  }

  async function handleLoadDetail(instanceId: string) {
    const requestId = ++detailRequestSeq.current
    setDetailLoading(true)
    setDetailError('')

    try {
      const data = await apiRequest<unknown>(`/api/cluster/instances/detail/${encodeURIComponent(instanceId)}`)
      if (!mountedRef.current || requestId !== detailRequestSeq.current) return
      setDetail(data)
    } catch (err) {
      if (!mountedRef.current || requestId !== detailRequestSeq.current) return
      setDetail(null)
      setDetailError(getErrorMessage(err, 'operations.instances.detailFailed'))
    } finally {
      if (mountedRef.current && requestId === detailRequestSeq.current) setDetailLoading(false)
    }
  }

  return (
    <Stack spacing={3}>
      <Box>
        <Typography variant="h4" component="h1">
          {t('operations.instances.title')}
        </Typography>
        <Typography color="text.secondary">{t('operations.instances.description')}</Typography>
      </Box>

      {error.message && <ErrorState message={isOperationsMessageKey(error.message) ? t(error.message) : error.message} eventKey={error.eventKey} />}
      {filterError && <ErrorState message={isOperationsMessageKey(filterError) ? t(filterError) : filterError} />}

      <Paper component="form" onSubmit={(event) => void handleFilter(event)} sx={{ p: 3 }}>
        <Stack spacing={2}>
          <Typography variant="h6" component="h2">
            {t('operations.instances.filterByElement')}
          </Typography>
          <Stack data-testid="instances-filter-row" direction={{ xs: 'column', md: 'row' }} spacing={2} alignItems={{ xs: 'stretch', md: 'flex-start' }}>
            <Autocomplete
              freeSolo
              options={appOptions}
              value={app}
              inputValue={app}
              loading={candidateLoading}
              onInputChange={(_, value, reason) => {
                if (reason === 'input') {
                  setApp(value)
                  setEnv('')
                  setKey('')
                  setEnvOptions([])
                  setKeyOptions([])
                }
              }}
              onChange={(_, value) => {
                const nextApp = value || ''
                setApp(nextApp)
                setEnv('')
                setKey('')
                setKeyOptions([])
                void loadEnvCandidates(nextApp)
              }}
              renderInput={(params) => <TextField {...params} label={t('operations.app')} helperText={t('operations.instances.appHelp')} />}
              sx={{ flex: 1 }}
              fullWidth
            />
            <Autocomplete
              freeSolo
              options={envOptions}
              value={env}
              inputValue={env}
              loading={candidateLoading}
              disabled={!app}
              onInputChange={(_, value, reason) => {
                if (reason === 'input') {
                  setEnv(value)
                  setKey('')
                  setKeyOptions([])
                }
              }}
              onChange={(_, value) => {
                const nextEnv = value || ''
                setEnv(nextEnv)
                setKey('')
                void loadKeyCandidates(app, nextEnv)
              }}
              renderInput={(params) => <TextField {...params} label={t('operations.env')} helperText={t('operations.instances.envHelp')} />}
              sx={{ flex: 1 }}
              fullWidth
            />
            <Autocomplete
              freeSolo
              options={keyOptions}
              value={key}
              inputValue={key}
              loading={candidateLoading}
              disabled={!app || !env}
              onInputChange={(_, value, reason) => {
                if (reason === 'input') setKey(value)
              }}
              onChange={(_, value) => setKey(value || '')}
              renderInput={(params) => <TextField {...params} label={t('operations.key')} helperText={t('operations.instances.keyHelp')} />}
              sx={{ flex: 1 }}
              fullWidth
            />
            <Stack direction="row" spacing={1.5} sx={{ whiteSpace: 'nowrap' }}>
              <Button type="submit" variant="contained" disabled={loading} sx={{ height: 40 }}>
                {t('operations.instances.filter')}
              </Button>
              <Button variant="outlined" onClick={() => void handleRefreshAll()} disabled={loading} sx={{ height: 40 }}>
                {t('operations.instances.refreshAll')}
              </Button>
            </Stack>
          </Stack>
        </Stack>
      </Paper>

      {loading ? (
        <LoadingState label={t('operations.instances.loading')} />
      ) : error.message ? null : instances.length === 0 ? (
        <EmptyState title={t('operations.instances.emptyTitle')} description={t('operations.instances.emptyDescription')} />
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>{t('operations.instances.client')}</TableCell>
                <TableCell>{t('operations.instances.agent')}</TableCell>
                <TableCell>{t('operations.instances.ip')}</TableCell>
                <TableCell>{t('operations.targets')}</TableCell>
                <TableCell>{t('operations.instances.lastSeen')}</TableCell>
                <TableCell align="right">{t('operations.instances.actions')}</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {instances.map((instance, index) => {
                const instanceId = instance.id || instance.clientId
                const rowKey = `${instanceId || 'unknown'}-${index}`

                return (
                  <TableRow key={rowKey} hover>
                    <TableCell>{instance.clientId || '-'}</TableCell>
                    <TableCell>{instance.agentId || '-'}</TableCell>
                    <TableCell>{instance.clientIp || '-'}</TableCell>
                    <TableCell>{renderTargets(instance)}</TableCell>
                    <TableCell title={instance.lastRenewTimestamp ? formatDateTime(instance.lastRenewTimestamp * 1000) : undefined}>{formatLastSeen(instance.lastRenewTimestamp, t)}</TableCell>
                    <TableCell align="right">
                      <Button size="small" onClick={() => instanceId && void handleLoadDetail(instanceId)} disabled={detailLoading || !instanceId}>
                        {t('operations.instances.detail')}
                      </Button>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {(detailLoading || Boolean(detailError) || detail !== null) && (
        <Paper sx={{ p: 3 }}>
          <Stack spacing={2}>
            <Typography variant="h6" component="h2">
              {t('operations.instances.detailTitle')}
            </Typography>
            {detailError && <ErrorState message={isOperationsMessageKey(detailError) ? t(detailError) : detailError} />}
            {detailLoading ? (
              <LoadingState label={t('operations.instances.loadingDetail')} />
            ) : detail ? (
              <Box
                component="pre"
                sx={{
                  m: 0,
                  p: 2,
                  overflowX: 'auto',
                  bgcolor: 'grey.100',
                  borderRadius: 1,
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                }}
              >
                {JSON.stringify(detail, null, 2)}
              </Box>
            ) : null}
          </Stack>
        </Paper>
      )}
    </Stack>
  )
}

export function InstancesPage() {
  const [searchParams] = useSearchParams()
  const initialApp = searchParams.get('app') || ''
  const initialEnv = searchParams.get('env') || ''
  const initialKey = searchParams.get('key') || ''

  return <InstancesPageFlow key={`${initialApp}/${initialEnv}/${initialKey}`} initialApp={initialApp} initialEnv={initialEnv} initialKey={initialKey} />
}
