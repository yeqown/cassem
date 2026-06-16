import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import DeviceHubIcon from '@mui/icons-material/DeviceHub'
import DnsIcon from '@mui/icons-material/Dns'
import HubIcon from '@mui/icons-material/Hub'
import StorageIcon from '@mui/icons-material/Storage'
import {
  Box,
  Button,
  Chip,
  Divider,
  Paper,
  Stack,
  Typography,
} from '@mui/material'
import { alpha } from '@mui/material/styles'
import { EmptyState, ErrorState, LoadingState } from '../../components/StateView'
import type { AgentNode, ClusterTopologyResponse, CommonResponse, DBNode, HealthState, Instance } from '../../domain/types'
import { ApiError } from '../../lib/api'
import { clearSession, getSession } from '../../lib/session'

const INSTANCE_AGGREGATION_THRESHOLD = 5
const EMPTY_DBS: DBNode[] = []
const EMPTY_AGENTS: AgentNode[] = []
const EMPTY_INSTANCES: Instance[] = []

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback
}

function normalizeTopologyData(data?: ClusterTopologyResponse | AgentNode[]) {
  if (Array.isArray(data)) return { dbs: [], agents: data, instances: [] }

  return {
    dbs: data?.dbs || [],
    agents: data?.agents || [],
    instances: data?.instances || [],
  }
}

async function requestTopology() {
  const headers: Record<string, string> = { Accept: 'application/json' }
  const session = getSession()
  if (session) headers['X-CASSEM-SESSION'] = session

  const response = await fetch('/api/cluster/topology', { headers })
  const payload = await response.json().catch(() => null) as
    | AgentNode[]
    | ClusterTopologyResponse
    | CommonResponse<ClusterTopologyResponse | AgentNode[] | { agents?: AgentNode[] }>
    | null

  if (Array.isArray(payload)) return normalizeTopologyData(payload)
  if (payload && !('errcode' in payload)) return normalizeTopologyData(payload as ClusterTopologyResponse)

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
  if (Array.isArray(data)) return normalizeTopologyData(data)
  return normalizeTopologyData(data as ClusterTopologyResponse)
}

function extractHost(addr?: string) {
  const value = addr?.trim()
  if (!value) return ''

  try {
    const parsed = new URL(value)
    return parsed.hostname || value
  } catch {
    const withoutScheme = value.replace(/^\w+:\/\//, '')
    return withoutScheme.includes(':') ? withoutScheme.split(':')[0] : withoutScheme
  }
}

function normalizeHealth(health?: HealthState, fallback: HealthState = 'offline') {
  if (health === 'healthy' || health === 'unhealthy' || health === 'offline') return health
  return fallback
}

function healthLabel(health: HealthState) {
  switch (health) {
    case 'healthy':
      return 'Healthy'
    case 'unhealthy':
      return 'Unhealthy'
    case 'offline':
      return 'Offline'
  }
}

function healthColor(health: HealthState) {
  switch (health) {
    case 'healthy':
      return 'success'
    case 'unhealthy':
      return 'warning'
    case 'offline':
      return 'default'
  }
}

function nodeTestId(kind: string, id: string) {
  const normalized = id.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
  if (!normalized) return `topology-node-${kind}-unknown`
  if (normalized.startsWith(`${kind}-`)) return `topology-node-${normalized}`
  return `topology-node-${kind}-${normalized}`
}

type TopologyNodeProps = {
  kind: 'db' | 'agent' | 'instance' | 'group'
  id: string
  ip?: string
  addr?: string
  health: HealthState
  subtitle?: string
  aggregateCount?: number
}

function NodeIcon({ kind }: Pick<TopologyNodeProps, 'kind'>) {
  if (kind === 'db') return <StorageIcon fontSize="small" />
  if (kind === 'agent') return <HubIcon fontSize="small" />
  if (kind === 'group') return <DeviceHubIcon fontSize="small" />
  return <DnsIcon fontSize="small" />
}

function TopologyNode({ kind, id, ip, addr, health, subtitle, aggregateCount }: TopologyNodeProps) {
  const healthTone = health === 'healthy' ? 'success.main' : health === 'unhealthy' ? 'warning.main' : 'text.disabled'

  return (
    <Paper
      variant="outlined"
      data-testid={nodeTestId(kind === 'group' ? 'instance-group' : kind, id)}
      sx={{
        p: 2,
        borderColor: (theme) => alpha(theme.palette.divider, 0.9),
        bgcolor: (theme) => health === 'offline' ? alpha(theme.palette.grey[500], 0.08) : 'background.paper',
        boxShadow: (theme) => `inset 4px 0 0 ${health === 'offline' ? theme.palette.grey[400] : health === 'unhealthy' ? theme.palette.warning.main : theme.palette.success.main}`,
      }}
    >
      <Stack spacing={1.25}>
        <Stack direction="row" spacing={1} alignItems="center" justifyContent="space-between">
          <Stack direction="row" spacing={1} alignItems="center" sx={{ minWidth: 0 }}>
            <Box
              sx={{
                width: 30,
                height: 30,
                borderRadius: '50%',
                display: 'grid',
                placeItems: 'center',
                color: healthTone,
                bgcolor: (theme) => alpha(health === 'healthy' ? theme.palette.success.main : health === 'unhealthy' ? theme.palette.warning.main : theme.palette.grey[500], 0.12),
              }}
            >
              <NodeIcon kind={kind} />
            </Box>
            <Box sx={{ minWidth: 0 }}>
              <Typography variant="subtitle2" noWrap>{aggregateCount ? `${aggregateCount} instances` : id || '-'}</Typography>
              {subtitle && <Typography variant="caption" color="text.secondary" noWrap>{subtitle}</Typography>}
            </Box>
          </Stack>
          <Chip size="small" color={healthColor(health)} label={healthLabel(health)} />
        </Stack>
        <Divider />
        <Stack spacing={0.5}>
          <Typography variant="caption" color="text.secondary">ID: {id || '-'}</Typography>
          <Typography variant="caption" color="text.secondary">IP: {ip || '-'}</Typography>
          {addr && <Typography variant="caption" color="text.secondary">Addr: {addr}</Typography>}
        </Stack>
      </Stack>
    </Paper>
  )
}

type TopologyColumnProps = {
  title: string
  description: string
  testId: string
  children: ReactNode
}

function TopologyColumn({ title, description, testId, children }: TopologyColumnProps) {
  return (
    <Stack data-testid={testId} spacing={1.5}>
      <Box>
        <Typography variant="overline" color="text.secondary">{title}</Typography>
        <Typography variant="body2" color="text.secondary">{description}</Typography>
      </Box>
      {children}
    </Stack>
  )
}

type TopologyLinkProps = {
  testId: string
  label: string
}

function TopologyLink({ testId, label }: TopologyLinkProps) {
  return (
    <Box
      data-testid={testId}
      aria-label={label}
      sx={{
        alignSelf: 'stretch',
        display: 'flex',
        alignItems: { xs: 'center', lg: 'flex-start' },
        justifyContent: 'center',
        minHeight: { xs: 28, lg: 180 },
        pt: { lg: 9.5 },
      }}
    >
      <Box
        sx={{
          position: 'relative',
          width: { xs: 2, lg: '100%' },
          height: { xs: 28, lg: 2 },
          bgcolor: (theme) => alpha(theme.palette.primary.main, 0.22),
          backgroundImage: (theme) => `linear-gradient(90deg, ${alpha(theme.palette.primary.main, 0.08)}, ${alpha(theme.palette.primary.main, 0.72)})`,
          backgroundSize: { xs: '2px 18px', lg: '28px 2px' },
          animation: 'topology-flow 1.4s linear infinite',
          '&::after': {
            content: '""',
            position: 'absolute',
            right: { lg: -2 },
            bottom: { xs: -2 },
            top: { lg: -4 },
            left: { xs: -4 },
            width: 10,
            height: 10,
            borderRadius: '50%',
            bgcolor: 'primary.main',
            boxShadow: (theme) => `0 0 0 4px ${alpha(theme.palette.primary.main, 0.12)}`,
          },
        }}
      />
    </Box>
  )
}

function groupInstancesByAgent(instances: Instance[]) {
  return instances.reduce<Record<string, Instance[]>>((acc, instance) => {
    const agentId = instance.agentId || 'unassigned'
    acc[agentId] = acc[agentId] || []
    acc[agentId].push(instance)
    return acc
  }, {})
}

function renderInstanceNodes(instances: Instance[]) {
  const grouped = groupInstancesByAgent(instances)

  return Object.entries(grouped).flatMap(([agentId, agentInstances]) => {
    if (agentInstances.length > INSTANCE_AGGREGATION_THRESHOLD) {
      return [
        <TopologyNode
          key={`group-${agentId}`}
          kind="group"
          id={agentId}
          ip="-"
          health="healthy"
          subtitle={`Attached to ${agentId}`}
          aggregateCount={agentInstances.length}
        />,
      ]
    }

    return agentInstances.map((instance, index) => {
      const id = instance.clientId || `${agentId}-${index + 1}`
      const health = normalizeHealth(instance.health, instance.lastRenewTimestamp ? 'healthy' : 'offline')
      return (
        <TopologyNode
          key={`${id}-${instance.clientIp || index}`}
          kind="instance"
          id={id}
          ip={instance.clientIp || '-'}
          health={health}
          subtitle={`Attached to ${agentId}`}
        />
      )
    })
  })
}

function isTopologyEmpty(topology: ClusterTopologyResponse) {
  return (topology.dbs || []).length === 0 && (topology.agents || []).length === 0 && (topology.instances || []).length === 0
}

export function AgentsPage() {
  const [topology, setTopology] = useState<ClusterTopologyResponse>({ dbs: [], agents: [], instances: [] })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const requestSeq = useRef(0)
  const mountedRef = useRef(false)
  const lastLoadKeyRef = useRef('')

  const loadTopology = useCallback(async () => {
    const requestId = ++requestSeq.current
    setLoading(true)

    try {
      const data = await requestTopology()
      if (!mountedRef.current || requestId !== requestSeq.current) return
      setTopology(data)
      setError('')
    } catch (err) {
      if (!mountedRef.current || requestId !== requestSeq.current) return
      setTopology({ dbs: [], agents: [], instances: [] })
      setError(getErrorMessage(err, 'failed to load cluster topology'))
    } finally {
      if (mountedRef.current && requestId === requestSeq.current) setLoading(false)
    }
  }, [])

  useEffect(() => {
    mountedRef.current = true

    const loadKey = 'cluster-topology'
    if (lastLoadKeyRef.current !== loadKey) {
      lastLoadKeyRef.current = loadKey
      queueMicrotask(() => {
        void loadTopology()
      })
    }

    return () => {
      mountedRef.current = false
    }
  }, [loadTopology])

  const dbs = topology.dbs || EMPTY_DBS
  const agents = topology.agents || EMPTY_AGENTS
  const instances = topology.instances || EMPTY_INSTANCES
  const instanceNodes = useMemo(() => renderInstanceNodes(instances), [instances])

  return (
    <Stack spacing={3}>
      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} justifyContent="space-between" alignItems={{ md: 'center' }}>
        <Box>
          <Typography variant="h4" component="h1">
            Agents topology
          </Typography>
          <Typography color="text.secondary">Inspect db, agent, and client instance relationships with live health signals.</Typography>
        </Box>
        <Button variant="outlined" onClick={() => void loadTopology()} disabled={loading}>
          Refresh
        </Button>
      </Stack>

      {error && <ErrorState message={error} />}

      {loading ? (
        <LoadingState label="Loading cluster topology" />
      ) : error ? null : isTopologyEmpty(topology) ? (
        <EmptyState title="No topology nodes found" description="No db, agent, or instance nodes were returned by the backend." />
      ) : (
        <Paper
          sx={{
            p: 3,
            '@keyframes topology-flow': {
              from: { backgroundPositionX: 0 },
              to: { backgroundPositionX: 36 },
            },
          }}
        >
          <Stack spacing={3} sx={{ position: 'relative' }}>
            <Box>
              <Typography variant="h6" component="h2">
                Cluster topology
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Animated view from cassemdb through agents to connected instances.
              </Typography>
            </Box>

            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              <Chip label={`${dbs.length} dbs`} color="primary" variant="outlined" />
              <Chip label={`${agents.length} agents`} color="primary" variant="outlined" />
              <Chip label={`${instances.length} instances`} color="primary" variant="outlined" />
            </Stack>

            <Box
              sx={{
                display: 'grid',
                gridTemplateColumns: { xs: '1fr', lg: 'minmax(220px, 1fr) 72px minmax(220px, 1fr) 72px minmax(220px, 1fr)' },
                gap: { xs: 2, lg: 1.5 },
                alignItems: 'start',
              }}
            >
              <TopologyColumn title="DBs" description="Configured cassemdb endpoints." testId="topology-dbs">
                {dbs.map((db: DBNode, index) => {
                  const id = db.id || `db-${index + 1}`
                  const health = normalizeHealth(db.health)
                  const ip = db.ip || extractHost(db.addr)
                  return <TopologyNode key={`${id}-${db.addr || index}`} kind="db" id={id} ip={ip} addr={db.addr} health={health} />
                })}
              </TopologyColumn>

              <TopologyLink testId="topology-link-dbs-agents" label="DBs connect to agents" />

              <TopologyColumn title="Agents" description="Registered edge cache nodes." testId="topology-agents">
                {agents.map((agent, index) => {
                  const id = agent.agentId || `agent-${index + 1}`
                  const health = normalizeHealth(agent.health, agent.addr ? 'healthy' : 'offline')
                  const ip = agent.ip || extractHost(agent.addr)
                  return <TopologyNode key={`${id}-${agent.addr || index}`} kind="agent" id={id} ip={ip} addr={agent.addr} health={health} />
                })}
              </TopologyColumn>

              <TopologyLink testId="topology-link-agents-instances" label="Agents connect to instances" />

              <TopologyColumn title="Instances" description="Client instances grouped by agent." testId="topology-instances">
                {instanceNodes.length === 0 ? (
                  <Typography color="text.secondary">No client instances.</Typography>
                ) : instanceNodes}
              </TopologyColumn>
            </Box>
          </Stack>
        </Paper>
      )}
    </Stack>
  )
}
