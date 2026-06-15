import type { AgentNode, Element, ElementsResponse, InstancesResponse } from '../../domain/types'
import { apiRequest, buildQuery } from '../../lib/api'

export type VersionOption = {
  value: string
  label: string
  version: number
  published: boolean
}

function uniqueStrings(values: string[]) {
  return Array.from(new Set(values.filter(Boolean)))
}

export function toVersionOptions(elements: Element[]) {
  return elements
    .flatMap((element) => {
      if (typeof element.version !== 'number' || !Number.isFinite(element.version) || element.version <= 0) return []
      return [{
        value: String(element.version),
        label: `v${element.version} ${element.published ? 'published' : 'draft'}`,
        version: element.version,
        published: Boolean(element.published),
      }]
    })
    .sort((left, right) => right.version - left.version)
}

export function getUsingVersion(elements: Element[]) {
  for (const element of elements) {
    const usingVersion = element.metadata?.usingVersion
    if (typeof usingVersion === 'number' && Number.isFinite(usingVersion) && usingVersion > 0) return usingVersion
  }

  return null
}

export async function requestVersionOptions(appId: string, env: string, key: string) {
  const data = await apiRequest<ElementsResponse>(
    `/api/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}/versions${buildQuery({ limit: 100 })}`,
  )
  const elements = data.elements || []
  return { options: toVersionOptions(elements), usingVersion: getUsingVersion(elements) }
}

export async function requestAgentIdOptions() {
  const data = await apiRequest<{ agents?: AgentNode[] }>(`/api/cluster/agents${buildQuery({ limit: 100 })}`)
  return uniqueStrings((data.agents || []).map((agent) => agent.agentId || ''))
}

export async function requestInstanceIdOptions(appId: string, env: string, key: string) {
  const data = await apiRequest<InstancesResponse>(`/api/cluster/instances/filter${buildQuery({ app: appId, env, key })}`)
  return uniqueStrings((data.instances || []).map((instance) => instance.clientId || ''))
}
