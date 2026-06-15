export function resolveRouterBasename(baseUrl: string) {
  if (!baseUrl || baseUrl === '/') return '/'
  return baseUrl.endsWith('/') ? baseUrl.slice(0, -1) : baseUrl
}
