import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import DeviceHubIcon from '@mui/icons-material/DeviceHub'
import DnsIcon from '@mui/icons-material/Dns'
import HubIcon from '@mui/icons-material/Hub'
import StorageIcon from '@mui/icons-material/Storage'
import {
  Box,
  Button,
  Chip,
  Paper,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material'
import { alpha } from '@mui/material/styles'
import { EmptyState, ErrorState, LoadingState } from '../../components/StateView'
import { useErrorState } from '../../components/useErrorState'
import type { AgentNode, ClusterTopologyResponse, CommonResponse, DBNode, HealthState, Instance } from '../../domain/types'
import { useI18n, type Translator } from '../../i18n'
import { isOperationsMessageKey } from '../../i18n/catalogs/operations'
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

function healthLabel(health: HealthState, t: Translator) {
  switch (health) {
    case 'healthy':
      return t('operations.topology.healthy')
    case 'unhealthy':
      return t('operations.topology.unhealthy')
    case 'offline':
      return t('operations.topology.offline')
  }
}

function nodeTestId(kind: string, id: string) {
  const normalized = id.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
  if (!normalized) return `topology-node-${kind}-unknown`
  if (normalized.startsWith(`${kind}-`)) return `topology-node-${normalized}`
  return `topology-node-${kind}-${normalized}`
}

type TopologyNodeKind = 'db' | 'agent' | 'instance' | 'group'

type TopologyNodeProps = {
  kind: TopologyNodeKind
  id: string
  ip?: string
  addr?: string
  health: HealthState
  subtitle?: string
  aggregateCount?: number
  attachedAgentId?: string
}

type TopologyGraphNode = TopologyNodeProps & {
  graphId: string
  edgeToken: string
  layer: 'dbs' | 'agents' | 'instances'
  x: number
  y: number
}

type TopologyEdge = {
  key: string
  from: Pick<TopologyGraphNode, 'x' | 'y'>
  to: Pick<TopologyGraphNode, 'x' | 'y'>
  testId: string
  label: string
}

type TopologyGraphModel = {
  nodes: TopologyGraphNode[]
  edges: TopologyEdge[]
  height: number
}

function normalizeGraphToken(id: string) {
  return id.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'unknown'
}

function edgeToken(kind: TopologyNodeKind, id: string) {
  const normalized = normalizeGraphToken(id)
  if (kind === 'group') return normalized.startsWith('instance-group-') ? normalized : `instance-group-${normalized}`
  if (kind === 'instance') return normalized.startsWith('instance-') ? normalized : `instance-${normalized}`
  return normalized
}

function layerY(index: number, total: number) {
  if (total <= 1) return 50
  return 16 + (68 * index) / (total - 1)
}

function positionLayer(nodes: TopologyNodeProps[], layer: TopologyGraphNode['layer'], x: number) {
  return nodes.map<TopologyGraphNode>((node, index) => ({
    ...node,
    graphId: `${layer}-${edgeToken(node.kind, node.id)}-${index}`,
    edgeToken: edgeToken(node.kind, node.id),
    layer,
    x,
    y: layerY(index, nodes.length),
  }))
}

function NodeIcon({ kind }: Pick<TopologyNodeProps, 'kind'>) {
  if (kind === 'db') return <StorageIcon fontSize="small" />
  if (kind === 'agent') return <HubIcon fontSize="small" />
  if (kind === 'group') return <DeviceHubIcon fontSize="small" />
  return <DnsIcon fontSize="small" />
}

function TopologyNode({ kind, id, ip, addr, health, subtitle, aggregateCount }: TopologyNodeProps) {
  const { t, formatNumber } = useI18n()
  const healthTone = health === 'healthy' ? 'success.main' : health === 'unhealthy' ? 'warning.main' : 'text.disabled'
  const accent = health === 'healthy' ? 'success.main' : health === 'unhealthy' ? 'warning.main' : 'grey.400'
  const label = aggregateCount ? t('operations.topology.instanceCount', { count: formatNumber(aggregateCount) }) : id || '-'
  const typeLabel = t(kind === 'db' ? 'operations.topology.db' : kind === 'agent' ? 'operations.topology.agent' : kind === 'group' ? 'operations.topology.instances' : 'operations.topology.instance')

  return (
    <Tooltip
      arrow
      title={(
        <Stack spacing={0.25}>
          <Typography variant="caption">{t('operations.topology.type', { type: typeLabel })}</Typography>
          <Typography variant="caption">{t('operations.topology.id', { id: id || '-' })}</Typography>
          <Typography variant="caption">{t('operations.topology.health', { health: healthLabel(health, t) })}</Typography>
          {subtitle && <Typography variant="caption">{subtitle}</Typography>}
          <Typography variant="caption">{t('operations.topology.ip', { ip: ip || '-' })}</Typography>
          {addr && <Typography variant="caption">{t('operations.topology.addr', { addr })}</Typography>}
        </Stack>
      )}
    >
      <Paper
        variant="outlined"
        data-testid={nodeTestId(kind === 'group' ? 'instance-group' : kind, id)}
        sx={{
          width: { xs: 96, md: 104 },
          height: { xs: 96, md: 104 },
          p: 1,
          borderRadius: 0,
          display: 'grid',
          placeItems: 'center',
          textAlign: 'center',
          borderColor: accent,
          bgcolor: 'background.paper',
          backgroundImage: (theme) => `radial-gradient(circle at 50% 18%, ${alpha(health === 'healthy' ? theme.palette.success.main : health === 'unhealthy' ? theme.palette.warning.main : theme.palette.grey[500], 0.14)}, transparent 48%)`,
          boxShadow: (theme) => `0 10px 26px ${alpha(theme.palette.common.black, 0.10)}, inset 0 0 0 4px ${alpha(health === 'healthy' ? theme.palette.success.main : health === 'unhealthy' ? theme.palette.warning.main : theme.palette.grey[500], 0.12)}`,
        }}
      >
        <Stack spacing={0.35} alignItems="center" sx={{ minWidth: 0, width: '100%' }}>
          <Box
            sx={{
              width: 24,
              height: 24,
              borderRadius: 0,
              display: 'grid',
              placeItems: 'center',
              color: healthTone,
              bgcolor: (theme) => alpha(health === 'healthy' ? theme.palette.success.main : health === 'unhealthy' ? theme.palette.warning.main : theme.palette.grey[500], 0.14),
              '& .MuiSvgIcon-root': { fontSize: 18 },
            }}
          >
            <NodeIcon kind={kind} />
          </Box>
          <Typography variant="caption" color="text.secondary" sx={{ fontSize: 11, lineHeight: 1 }}>{typeLabel}</Typography>
          <Typography variant="subtitle2" noWrap sx={{ maxWidth: '100%', fontSize: 14, lineHeight: 1.1 }}>{label}</Typography>
          <Box
            aria-label={healthLabel(health, t)}
            sx={{
              width: 10,
              height: 10,
              borderRadius: 0,
              bgcolor: accent,
              boxShadow: (theme) => `0 0 0 4px ${alpha(health === 'healthy' ? theme.palette.success.main : health === 'unhealthy' ? theme.palette.warning.main : theme.palette.grey[500], 0.14)}`,
            }}
          />
        </Stack>
      </Paper>
    </Tooltip>
  )
}

function TopologyGraph({ graph }: { graph: TopologyGraphModel }) {
  const dbNodes = graph.nodes.filter((node) => node.layer === 'dbs')
  const agentNodes = graph.nodes.filter((node) => node.layer === 'agents')
  const instanceNodes = graph.nodes.filter((node) => node.layer === 'instances')

  function renderNode(node: TopologyGraphNode) {
    return (
      <Box
        key={node.graphId}
        sx={{
          position: 'absolute',
          left: `${node.x}%`,
          top: `${node.y}%`,
          transform: 'translate(-50%, -50%)',
          zIndex: 1,
        }}
      >
        <TopologyNode {...node} />
      </Box>
    )
  }

  return (
    <Box
      data-testid="topology-graph"
      sx={{
        minWidth: { xs: 760, lg: 'auto' },
        p: { xs: 3, md: 4 },
        overflow: 'hidden',
        borderRadius: 4,
        border: 1,
        borderColor: 'divider',
        bgcolor: (theme) => alpha(theme.palette.primary.main, 0.035),
        backgroundImage: (theme) => `radial-gradient(circle at 16% 20%, ${alpha(theme.palette.primary.main, 0.12)}, transparent 26%), radial-gradient(circle at 84% 80%, ${alpha(theme.palette.success.main, 0.10)}, transparent 28%)`,
      }}
    >
      <Box sx={{ position: 'relative', height: graph.height }}>
        <Box
          component="svg"
          viewBox="0 0 100 100"
          preserveAspectRatio="none"
          sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', zIndex: 0 }}
        >
          {graph.edges.map((edge) => (
            <line
              key={edge.key}
              data-testid={edge.testId}
              aria-label={edge.label}
              x1={edge.from.x}
              y1={edge.from.y}
              x2={edge.to.x}
              y2={edge.to.y}
              vectorEffect="non-scaling-stroke"
              stroke="currentColor"
              strokeWidth={2}
              strokeDasharray="10 8"
              strokeLinecap="round"
              style={{ color: 'rgba(25, 118, 210, 0.48)', animation: 'topology-edge-flow 1.2s linear infinite' }}
            />
          ))}
        </Box>
        <Box data-testid="topology-dbs" sx={{ display: 'contents' }}>{dbNodes.map(renderNode)}</Box>
        <Box data-testid="topology-agents" sx={{ display: 'contents' }}>{agentNodes.map(renderNode)}</Box>
        <Box data-testid="topology-instances" sx={{ display: 'contents' }}>{instanceNodes.map(renderNode)}</Box>
      </Box>
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

function getInstanceGraphNodes(instances: Instance[], t: Translator) {
  const grouped = groupInstancesByAgent(instances)

  return Object.entries(grouped).flatMap<TopologyNodeProps>(([agentId, agentInstances]) => {
    if (agentInstances.length > INSTANCE_AGGREGATION_THRESHOLD) {
      return [{
        kind: 'group',
        id: agentId,
        ip: '-',
        health: 'healthy',
        subtitle: t('operations.topology.attachedTo', { agentId: agentInstances[0].agentId || t('operations.topology.unassigned') }),
        aggregateCount: agentInstances.length,
        attachedAgentId: agentId,
      }]
    }

    return agentInstances.map((instance, index) => {
      const id = instance.clientId || `${agentId}-${index + 1}`
      return {
        kind: 'instance',
        id,
        ip: instance.clientIp || '-',
        health: normalizeHealth(instance.health, instance.lastRenewTimestamp ? 'healthy' : 'offline'),
        subtitle: t('operations.topology.attachedTo', { agentId: instance.agentId || t('operations.topology.unassigned') }),
        attachedAgentId: agentId,
      }
    })
  })
}

function buildTopologyGraph(dbs: DBNode[], agents: AgentNode[], instances: Instance[], t: Translator): TopologyGraphModel {
  const dbGraphNodes = positionLayer(dbs.map<TopologyNodeProps>((db, index) => {
    const id = db.id || `db-${index + 1}`
    return {
      kind: 'db',
      id,
      ip: db.ip || extractHost(db.addr),
      addr: db.addr,
      health: normalizeHealth(db.health),
    }
  }), 'dbs', 15)
  const agentGraphNodes = positionLayer(agents.map<TopologyNodeProps>((agent, index) => {
    const id = agent.agentId || `agent-${index + 1}`
    return {
      kind: 'agent',
      id,
      ip: agent.ip || extractHost(agent.addr),
      addr: agent.addr,
      health: normalizeHealth(agent.health, agent.addr ? 'healthy' : 'offline'),
    }
  }), 'agents', 50)
  const instanceGraphNodes = positionLayer(getInstanceGraphNodes(instances, t), 'instances', 85)
  const edges: TopologyEdge[] = []

  dbGraphNodes.forEach((dbNode) => {
    agentGraphNodes.forEach((agentNode) => {
      edges.push({
        key: `${dbNode.graphId}-${agentNode.graphId}`,
        from: dbNode,
        to: agentNode,
        testId: `topology-edge-${dbNode.edgeToken}-${agentNode.edgeToken}`,
        label: t('operations.topology.connection', { from: dbNode.id, to: agentNode.id }),
      })
    })
  })

  const agentNodeById = new Map(agentGraphNodes.map((node) => [node.id, node]))
  instanceGraphNodes.forEach((instanceNode) => {
    const agentNode = agentNodeById.get(instanceNode.attachedAgentId || '') || agentGraphNodes[0]
    if (!agentNode) return
    const label = instanceNode.aggregateCount
      ? t('operations.topology.instanceConnection', { from: agentNode.id, count: instanceNode.aggregateCount })
      : t('operations.topology.connection', { from: agentNode.id, to: instanceNode.id })
    edges.push({
      key: `${agentNode.graphId}-${instanceNode.graphId}`,
      from: agentNode,
      to: instanceNode,
      testId: `topology-edge-${agentNode.edgeToken}-${instanceNode.edgeToken}`,
      label,
    })
  })

  const maxLayerSize = Math.max(dbGraphNodes.length, agentGraphNodes.length, instanceGraphNodes.length)
  return {
    nodes: [...dbGraphNodes, ...agentGraphNodes, ...instanceGraphNodes],
    edges,
    height: Math.max(300, maxLayerSize * 116),
  }
}

function isTopologyEmpty(topology: ClusterTopologyResponse) {
  return (topology.dbs || []).length === 0 && (topology.agents || []).length === 0 && (topology.instances || []).length === 0
}

export function AgentsPage() {
  const { t, formatNumber } = useI18n()
  const [topology, setTopology] = useState<ClusterTopologyResponse>({ dbs: [], agents: [], instances: [] })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useErrorState()
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
      setError(getErrorMessage(err, 'operations.topology.loadFailed'))
    } finally {
      if (mountedRef.current && requestId === requestSeq.current) setLoading(false)
    }
  }, [setError])

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
  const graph = useMemo(() => buildTopologyGraph(dbs, agents, instances, t), [dbs, agents, instances, t])

  return (
    <Stack spacing={3}>
      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} justifyContent="space-between" alignItems={{ md: 'center' }}>
        <Box>
          <Typography variant="h4" component="h1">
            {t('operations.topology.title')}
          </Typography>
          <Typography color="text.secondary">{t('operations.topology.description')}</Typography>
        </Box>
        <Button variant="outlined" onClick={() => void loadTopology()} disabled={loading}>
          {t('operations.topology.refresh')}
        </Button>
      </Stack>

      {error.message && <ErrorState message={isOperationsMessageKey(error.message) ? t(error.message) : error.message} eventKey={error.eventKey} />}

      {loading ? (
        <LoadingState label={t('operations.topology.loading')} />
      ) : error.message ? null : isTopologyEmpty(topology) ? (
        <EmptyState title={t('operations.topology.emptyTitle')} description={t('operations.topology.emptyDescription')} />
      ) : (
        <Paper
          sx={{
            p: 3,
            '@keyframes topology-edge-flow': {
              from: { strokeDashoffset: 0 },
              to: { strokeDashoffset: -18 },
            },
          }}
        >
          <Stack spacing={3} sx={{ position: 'relative' }}>
            <Box>
              <Typography variant="h6" component="h2">
                {t('operations.topology.clusterTitle')}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                {t('operations.topology.clusterDescription')}
              </Typography>
            </Box>

            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
              <Chip label={t('operations.topology.dbCount', { count: formatNumber(dbs.length) })} color="primary" variant="outlined" />
              <Chip label={t('operations.topology.agentCount', { count: formatNumber(agents.length) })} color="primary" variant="outlined" />
              <Chip label={t('operations.topology.instanceCount', { count: formatNumber(instances.length) })} color="primary" variant="outlined" />
            </Stack>

            <Box sx={{ overflowX: 'auto', pb: 1 }}>
              <TopologyGraph graph={graph} />
            </Box>
          </Stack>
        </Paper>
      )}
    </Stack>
  )
}
