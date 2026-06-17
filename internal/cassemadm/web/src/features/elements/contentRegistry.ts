import { StreamLanguage, type LanguageSupport } from '@codemirror/language'
import { json } from '@codemirror/lang-json'
import { properties } from '@codemirror/legacy-modes/mode/properties'
import { toml } from '@codemirror/legacy-modes/mode/toml'
import type { Extension } from '@codemirror/state'
import { contentTypes } from '../../domain/types'

type ContentTooling = {
  label: string
  language?: Extension | LanguageSupport
}

function contentTypeKey(contentType?: number | string) {
  const numeric = Number(contentType)
  return Number.isFinite(numeric) ? numeric : undefined
}

const registry = new Map<number, ContentTooling>([
  [1, { label: 'JSON', language: json() }],
  [2, { label: 'TOML', language: StreamLanguage.define(toml) }],
  [3, { label: 'INI', language: StreamLanguage.define(properties) }],
  [4, { label: 'PLAINTEXT' }],
])

export function getContentTooling(contentType?: number | string) {
  const key = contentTypeKey(contentType)
  if (key !== undefined) return registry.get(key) || { label: String(contentType || '-') }

  return { label: String(contentType || '-') }
}

export function getContentTypeLabel(contentType?: number | string) {
  const key = contentTypeKey(contentType)
  return getContentTooling(contentType).label || contentTypes.find((item) => item.value === key)?.label || String(contentType || '-')
}

export function getContentLanguage(contentType?: number | string) {
  const language = getContentTooling(contentType).language
  return language ? [language] : []
}
