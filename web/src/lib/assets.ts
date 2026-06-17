export function assetUrl(path: string, baseUrl = import.meta.env.BASE_URL) {
  const assetPath = path.startsWith('/') ? path.slice(1) : path
  const base = !baseUrl || baseUrl === '/' ? '/' : baseUrl.endsWith('/') ? baseUrl : `${baseUrl}/`

  return `${base}${assetPath}`
}
