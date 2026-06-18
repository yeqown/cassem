import type { AgentNode, Element, ElementsResponse, InstancesResponse } from '../../domain/types'
import { apiRequest, buildQuery } from '../../lib/api'

export type VersionOption = {
  value: string
  label: string
  version: number
  published: boolean
  element: Element
}

function uniqueStrings(values: string[]) {
  return Array.from(new Set(values.filter(Boolean)))
}

export function toVersionOption(element: Element) {
  if (typeof element.version !== 'number' || !Number.isFinite(element.version) || element.version <= 0) return null

  return {
    value: String(element.version),
    label: `v${element.version} ${element.published ? 'published' : 'draft'}`,
    version: element.version,
    published: Boolean(element.published),
    element,
  }
}

export function toVersionOptions(elements: Element[]) {
  return elements
    .flatMap((element) => {
      const option = toVersionOption(element)
      return option ? [option] : []
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

export async function requestElement(appId: string, env: string, key: string) {
  return apiRequest<Element>(`/api/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}`)
}

export async function requestElementVersion(appId: string, env: string, key: string, version: number) {
  return apiRequest<Element>(
    `/api/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}${buildQuery({ version })}`,
  )
}

export async function requestComparisonElement(appId: string, env: string, key: string, version: number) {
  return requestElementVersion(appId, env, key, version)
}

export async function requestAgentIdOptions() {
  const data = await apiRequest<{ agents?: AgentNode[] }>(`/api/cluster/agents${buildQuery({ limit: 100 })}`)
  return uniqueStrings((data.agents || []).map((agent) => agent.agentId || ''))
}

export async function requestInstanceIdOptions(appId: string, env: string, key: string) {
  const data = await apiRequest<InstancesResponse>(`/api/cluster/instances/filter${buildQuery({ app: appId, env, key })}`)
  return uniqueStrings((data.instances || []).map((instance) => instance.clientId || ''))
}
