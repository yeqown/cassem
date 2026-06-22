import { jsonParseLinter } from '@codemirror/lang-json'
import { linter, type Diagnostic } from '@codemirror/lint'
import type { EditorState, Extension } from '@codemirror/state'
import { validateContent, type ContentDiagnostic, type ContentTypeInput } from './contentValidation'

function contentTypeKey(contentType?: ContentTypeInput) {
  const numeric = Number(contentType)
  if (Number.isFinite(numeric)) return numeric
  if (typeof contentType === 'string') {
    const normalized = contentType.toUpperCase()
    if (normalized === 'JSON') return 1
    if (normalized === 'TOML') return 2
    if (normalized === 'INI') return 3
    if (normalized === 'PLAINTEXT') return 4
  }
  return undefined
}

function diagnosticRange(state: EditorState, item: ContentDiagnostic) {
  const lineNumber = Math.max(1, Math.min(item.line || 1, state.doc.lines))
  const line = state.doc.line(lineNumber)
  const column = Math.max(1, item.column || 1)
  const from = Math.min(line.from + column - 1, line.to)
  const length = item.length || Math.max(1, line.to - from)
  const to = Math.min(line.to, from + length)

  return { from, to: Math.max(to, from + 1) }
}

export function createContentDiagnostics(contentType?: ContentTypeInput): Extension[] {
  if (contentTypeKey(contentType) === 1) {
    return [linter(jsonParseLinter(), { delay: 250 })]
  }

  return [
    linter((view): Diagnostic[] => {
      const result = validateContent(contentType, view.state.doc.toString())

      return (result.diagnostics || []).map((item): Diagnostic => ({
        ...diagnosticRange(view.state, item),
        severity: item.severity,
        message: item.message,
      }))
    }, { delay: 250 }),
  ]
}
