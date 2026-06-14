import { describe, expect, it } from 'vitest'
import { decodeRaw } from './raw'

describe('decodeRaw', () => {
  it('decodes base64 UTF-8 content', () => {
    expect(decodeRaw('eyJrZXkiOiAidmFsdWUifQ==')).toBe('{"key": "value"}')
  })

  it('returns original raw content when base64 decoding fails', () => {
    expect(decodeRaw('not base64 %')).toBe('not base64 %')
  })

  it('returns original raw content for invalid utf-8 after base64 decode', () => {
    expect(decodeRaw('////')).toBe('////')
  })
})
