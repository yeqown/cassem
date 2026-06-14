import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import {
  Box,
  Button,
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
import { useSearchParams } from 'react-router-dom'
import { EmptyState, ErrorState, LoadingState } from '../../components/StateView'
import type { Instance, InstancesResponse } from '../../domain/types'
import { ApiError, apiRequest, buildQuery } from '../../lib/api'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback
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
  const [app, setApp] = useState(initialApp)
  const [env, setEnv] = useState(initialEnv)
  const [key, setKey] = useState(initialKey)
  const [instances, setInstances] = useState<Instance[]>([])
  const [detail, setDetail] = useState<unknown>(null)
  const [loading, setLoading] = useState(true)
  const [detailLoading, setDetailLoading] = useState(false)
  const [error, setError] = useState('')
  const [detailError, setDetailError] = useState('')
  const [filterError, setFilterError] = useState('')
  const requestSeq = useRef(0)
  const detailRequestSeq = useRef(0)
  const mountedRef = useRef(false)

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
      setError(getErrorMessage(err, 'failed to load instances'))
    } finally {
      if (mountedRef.current && requestId === requestSeq.current) setLoading(false)
    }
  }, [])

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
        setError(getErrorMessage(err, 'failed to filter instances'))
      } finally {
        if (mountedRef.current && requestId === requestSeq.current) setLoading(false)
      }
    },
    [loadAll],
  )

  useEffect(() => {
    mountedRef.current = true

    queueMicrotask(() => {
      if (initialApp && initialEnv && initialKey) {
        void applyFilter(initialApp, initialEnv, initialKey)
      } else {
        void loadAll()
      }
    })

    return () => {
      mountedRef.current = false
    }
  }, [applyFilter, initialApp, initialEnv, initialKey, loadAll])

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
      setDetailError(getErrorMessage(err, 'failed to load instance detail'))
    } finally {
      if (mountedRef.current && requestId === detailRequestSeq.current) setDetailLoading(false)
    }
  }

  return (
    <Stack spacing={3}>
      <Box>
        <Typography variant="h4" component="h1">
          Instances
        </Typography>
        <Typography color="text.secondary">Inspect cluster clients, filter by element ownership, and load individual instance detail.</Typography>
      </Box>

      {error && <ErrorState message={error} />}
      {filterError && <ErrorState message={filterError} />}

      <Paper component="form" onSubmit={(event) => void handleFilter(event)} sx={{ p: 3 }}>
        <Stack spacing={2}>
          <Typography variant="h6" component="h2">
            Filter by element
          </Typography>
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <TextField label="App" value={app} onChange={(event) => setApp(event.target.value)} fullWidth helperText="Application name used by the watched element." />
            <TextField label="Env" value={env} onChange={(event) => setEnv(event.target.value)} fullWidth helperText="Environment name, for example prod." />
            <TextField label="Key" value={key} onChange={(event) => setKey(event.target.value)} fullWidth helperText="Exact element key watched by the instance." />
          </Stack>
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <Button type="submit" variant="contained" disabled={loading}>
              Filter
            </Button>
            <Button variant="outlined" onClick={() => void handleRefreshAll()} disabled={loading}>
              Refresh all
            </Button>
          </Stack>
        </Stack>
      </Paper>

      {loading ? (
        <LoadingState label="Loading instances" />
      ) : error ? null : instances.length === 0 ? (
        <EmptyState title="No instances found" description="Adjust the filter or refresh the full cluster instance list." />
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Client</TableCell>
                <TableCell>Agent</TableCell>
                <TableCell>IP</TableCell>
                <TableCell>App</TableCell>
                <TableCell>Env</TableCell>
                <TableCell>Key</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {instances.map((instance, index) => {
                const instanceId = instance.clientId
                const rowKey = `${instance.clientId || 'unknown'}-${index}`

                return (
                  <TableRow key={rowKey} hover>
                    <TableCell>{instance.clientId || '-'}</TableCell>
                    <TableCell>{instance.agentId || '-'}</TableCell>
                    <TableCell>{instance.clientIp || '-'}</TableCell>
                    <TableCell>{instance.app || '-'}</TableCell>
                    <TableCell>{instance.env || '-'}</TableCell>
                    <TableCell>{instance.key || '-'}</TableCell>
                    <TableCell align="right">
                      <Button size="small" onClick={() => instanceId && void handleLoadDetail(instanceId)} disabled={detailLoading || !instanceId}>
                        Detail
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
              Instance detail
            </Typography>
            {detailError && <ErrorState message={detailError} />}
            {detailLoading ? (
              <LoadingState label="Loading instance detail" />
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
