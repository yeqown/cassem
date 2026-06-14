export function decodeRaw(raw?: string) {
  if (!raw) return ''

  try {
    const binary = atob(raw)
    const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0))
    return new TextDecoder('utf-8', { fatal: true }).decode(bytes)
  } catch {
    return raw
  }
}
