import { parse as parseToml } from 'smol-toml'

export type ContentTypeInput = number | string | undefined

export type ContentDiagnostic = {
  message: string
  severity: 'error' | 'warning'
  line?: number
  column?: number
  length?: number
}

export type ContentValidationResult = {
  valid: boolean
  message?: string
  line?: number
  column?: number
  diagnostics?: ContentDiagnostic[]
}

type Validator = (raw: string) => ContentValidationResult

const validResult: ContentValidationResult = { valid: true }

const contentTypeNameKeys = new Map<string, number>([
  ['JSON', 1],
  ['TOML', 2],
  ['INI', 3],
  ['PLAINTEXT', 4],
])

function contentTypeKey(contentType?: ContentTypeInput) {
  const numeric = Number(contentType)
  if (Number.isFinite(numeric)) return numeric
  if (typeof contentType === 'string') return contentTypeNameKeys.get(contentType.toUpperCase())
  return undefined
}

function errorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

function invalidResult(message: string, line?: number, column?: number, length?: number): ContentValidationResult {
  const diagnostic: ContentDiagnostic = { message, severity: 'error', line, column, length }

  return { valid: false, message, line, column, diagnostics: [diagnostic] }
}

function validateJson(raw: string): ContentValidationResult {
  if (!raw.trim()) return validResult

  try {
    JSON.parse(raw)
    return validResult
  } catch (error) {
    return invalidResult(errorMessage(error, 'Invalid JSON'), 1, 1)
  }
}

function validateIni(raw: string): ContentValidationResult {
  const lines = raw.split(/\r?\n/)
  const sectionPattern = /^\[[^\]\r\n]+\]$/
  const keyValuePattern = /^[^=:#\r\n][^=:\r\n]*\s*[=:]\s*.*$/

  for (const [index, line] of lines.entries()) {
    const trimmed = line.trim()
    if (!trimmed || trimmed.startsWith(';') || trimmed.startsWith('#')) continue
    if (sectionPattern.test(trimmed) || keyValuePattern.test(trimmed)) continue

    return invalidResult('Expected key=value or section header', index + 1, 1, line.length || 1)
  }

  return validResult
}

function parserLine(error: unknown) {
  if (error && typeof error === 'object' && 'line' in error && typeof error.line === 'number') return error.line
  return undefined
}

function parserColumn(error: unknown) {
  if (error && typeof error === 'object' && 'column' in error && typeof error.column === 'number') return error.column + 1
  return undefined
}

function validateToml(raw: string): ContentValidationResult {
  if (!raw.trim()) return validResult

  try {
    parseToml(raw)
    return validResult
  } catch (error) {
    const line = parserLine(error)
    const column = parserColumn(error)
    return invalidResult(errorMessage(error, 'Invalid TOML'), line, column)
  }
}

const validators = new Map<number, Validator>([
  [1, validateJson],
  [2, validateToml],
  [3, validateIni],
])

export function validateContent(contentType: ContentTypeInput, raw: string): ContentValidationResult {
  const key = contentTypeKey(contentType)
  const validator = key === undefined ? undefined : validators.get(key)
  if (!validator) return validResult

  try {
    return validator(raw)
  } catch {
    return { valid: false, message: 'Unable to validate content' }
  }
}

export function formatValidationMessage(result: ContentValidationResult): string {
  if (result.valid || !result.message) return ''
  if (result.line !== undefined && result.column !== undefined) return `Line ${result.line}, Column ${result.column}: ${result.message}`
  if (result.line !== undefined) return `Line ${result.line}: ${result.message}`
  return result.message
}
