import { describe, expect, it } from 'vitest'
import { formatValidationMessage, validateContent } from './contentValidation'

describe('validateContent', () => {
  it('accepts valid JSON', () => {
    expect(validateContent(1, '{"enabled":true}')).toEqual({ valid: true })
  })

  it('rejects invalid JSON', () => {
    const result = validateContent('JSON', '{"enabled":')

    expect(result.valid).toBe(false)
    expect(result.message).toMatch(/json|unexpected|end/i)
  })

  it('accepts common INI syntax', () => {
    const result = validateContent(
      3,
      `; comment
# other comment

[server]
port=8080
host: localhost
rootKey=true`,
    )

    expect(result).toEqual({ valid: true })
  })

  it('rejects invalid INI lines with line numbers', () => {
    const result = validateContent('INI', 'this is not ini')

    expect(result).toMatchObject({
      valid: false,
      line: 1,
      message: 'Expected key=value or section header',
      diagnostics: [{ line: 1, column: 1, length: 15, message: 'Expected key=value or section header', severity: 'error' }],
    })
    expect(formatValidationMessage(result)).toBe('Line 1, Column 1: Expected key=value or section header')
  })

  it('accepts valid TOML', () => {
    expect(validateContent('TOML', 'title = "Demo"\n[server]\nport = 8080')).toEqual({ valid: true })
  })

  it('rejects invalid TOML with diagnostic location', () => {
    const result = validateContent(2, 'title =')

    expect(result.valid).toBe(false)
    expect(result.message).toBeTruthy()
    expect(result.diagnostics?.[0]).toMatchObject({ line: 1, column: 9, severity: 'error' })
  })

  it('keeps plaintext valid even when content looks malformed', () => {
    expect(validateContent(4, '{"enabled":')).toEqual({ valid: true })
    expect(validateContent('PLAINTEXT', 'this is not ini')).toEqual({ valid: true })
  })

  it('keeps unknown content types valid for forward compatibility', () => {
    expect(validateContent(99, 'anything')).toEqual({ valid: true })
    expect(validateContent('YAML', 'key: [')).toEqual({ valid: true })
  })

  it('formats messages with line and column', () => {
    expect(formatValidationMessage({ valid: false, line: 2, column: 4, message: 'bad token' })).toBe('Line 2, Column 4: bad token')
  })
})
