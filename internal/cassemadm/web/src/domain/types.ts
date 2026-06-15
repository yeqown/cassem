export type CommonResponse<T> = {
  errcode: number
  errmsg?: string
  data?: T
}

export type UserAccessBinding = {
  role: string
  domain: string
}

export type User = {
  account: string
  nickname?: string
  status?: number
  roles?: string[]
  bindingCount?: number
  accessSummary?: UserAccessBinding[]
}

export type LoginResponse = {
  user: User
  session: string
}

export type UsersResponse = {
  users?: User[]
}

export type UserAccessResponse = {
  bindings?: UserAccessBinding[]
}

export type DomainOptionsResponse = {
  domains?: string[]
}

export type AppMetadata = {
  id: string
  description?: string
  createdAt?: number
  creator?: string
  owner?: string
}

export type AppsResponse = {
  apps?: AppMetadata[]
}

export type EnvsResponse = {
  envs?: string[]
}

export type ElementMetadata = {
  key: string
  latestVersion?: number
  usingVersion?: number
  unpublishedVersion?: number
  contentType?: number | string
}

export type Element = {
  metadata: ElementMetadata
  raw?: string
  version?: number
  published?: boolean
}

export type ElementsResponse = {
  elements?: Element[]
}

export type ElementOperation = {
  operator?: string
  op?: string | number
  lastVersion?: number
  currentVersion?: number
  operatedAt?: number
}

export type ElementOperationsResponse = {
  operations?: ElementOperation[]
}

export type DiffResponse = {
  base?: Element
  compare?: Element
  diff: string
}

export type AgentNode = {
  agentId?: string
  addr?: string
}

export type Instance = {
  clientId?: string
  agentId?: string
  clientIp?: string
  app?: string
  env?: string
  key?: string
}

export type InstancesResponse = {
  instances?: Instance[]
}

export const contentTypes = [
  { value: 1, label: 'JSON' },
  { value: 2, label: 'TOML' },
  { value: 3, label: 'INI' },
  { value: 4, label: 'PLAINTEXT' },
] as const

export const roleOptions = [
  { value: 'superadmin', label: 'Super Admin' },
  { value: 'admin', label: 'Admin' },
  { value: 'appowner', label: 'App Owner' },
  { value: 'appdeveloper', label: 'Developer' },
] as const

export type RoleValue = (typeof roleOptions)[number]['value']
