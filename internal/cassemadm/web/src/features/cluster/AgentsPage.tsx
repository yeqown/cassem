import { useCallback, useEffect, useRef, useState } from 'react'
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
  Typography,
} from '@mui/material'
import { EmptyState, ErrorState, LoadingState } from '../../components/StateView'
import type { AgentNode, CommonResponse } from '../../domain/types'
import { ApiError, buildQuery } from '../../lib/api'
import { clearSession, getSession } from '../../lib/session'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback
}

async function requestAgents() {
  const headers: Record<string, string> = { Accept: 'application/json' }
  const session = getSession()
  if (session) headers['X-CASSEM-SESSION'] = session

  const response = await fetch(`/api/cluster/agents${buildQuery({ limit: 100 })}`, { headers })
  const payload = await response.json().catch(() => null) as AgentNode[] | CommonResponse<AgentNode[] | { agents?: AgentNode[] }> | null

  if (Array.isArray(payload)) return payload

  const auth =
    response.status === 401 ||
    payload?.errcode === 16 ||
    /unauthenticated|session expired|invalid session/i.test(payload?.errmsg || '')

  if (!response.ok || !payload || payload.errcode !== 0) {
    if (auth) {
      clearSession()
      window.location.assign('/ui/login')
    }
    throw new ApiError(payload?.errmsg || `HTTP ${response.status}`, payload?.errcode ?? -1, response.status, auth)
  }

  const data = payload.data
  return Array.isArray(data) ? data : data?.agents || []
}

export function AgentsPage() {
  const [agents, setAgents] = useState<AgentNode[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const requestSeq = useRef(0)
  const mountedRef = useRef(false)

  const loadAgents = useCallback(async () => {
    const requestId = ++requestSeq.current
    setLoading(true)

    try {
      const data = await requestAgents()
      if (!mountedRef.current || requestId !== requestSeq.current) return
      setAgents(data)
      setError('')
    } catch (err) {
      if (!mountedRef.current || requestId !== requestSeq.current) return
      setAgents([])
      setError(getErrorMessage(err, 'failed to load agents'))
    } finally {
      if (mountedRef.current && requestId === requestSeq.current) setLoading(false)
    }
  }, [])

  useEffect(() => {
    mountedRef.current = true

    queueMicrotask(() => {
      void loadAgents()
    })

    return () => {
      mountedRef.current = false
    }
  }, [loadAgents])

  return (
    <Stack spacing={3}>
      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} justifyContent="space-between" alignItems={{ md: 'center' }}>
        <Box>
          <Typography variant="h4" component="h1">
            Agents
          </Typography>
          <Typography color="text.secondary">Inspect known agent nodes and their advertised cluster addresses.</Typography>
        </Box>
        <Button variant="outlined" onClick={() => void loadAgents()} disabled={loading}>
          Refresh
        </Button>
      </Stack>

      {error && <ErrorState message={error} />}

      {loading ? (
        <LoadingState label="Loading agents" />
      ) : error ? null : agents.length === 0 ? (
        <EmptyState title="No agents found" description="No cluster agent nodes were returned by the backend." />
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Agent ID</TableCell>
                <TableCell>Address</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {agents.map((agent, index) => (
                <TableRow key={`${agent.agentId || 'unknown'}-${index}`} hover>
                  <TableCell>{agent.agentId || '-'}</TableCell>
                  <TableCell>{agent.addr || '-'}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </Stack>
  )
}
