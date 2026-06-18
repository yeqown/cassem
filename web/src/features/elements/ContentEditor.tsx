import CodeMirror from '@uiw/react-codemirror'
import { Chip, Stack } from '@mui/material'
import { getContentExtensions, getContentTypeLabel } from './contentRegistry'
import type { CodeTheme } from '../../lib/settings'

type ContentEditorProps = {
  value: string
  contentType?: number | string
  ariaLabel?: string
  codeTheme?: CodeTheme
  disabled?: boolean
  lineWrapping?: boolean
  minRows?: number
  maxRows?: number
  showContentType?: boolean
  onChange: (value: string) => void
}

export function ContentEditor({ value, contentType, ariaLabel = 'Raw content', codeTheme = 'github-light-plus', disabled = false, lineWrapping = true, minRows = 12, maxRows = 24, showContentType = true, onChange }: ContentEditorProps) {
  const minHeight = `${minRows * 24}px`
  const maxHeight = `${maxRows * 24}px`

  return (
    <Stack spacing={1.5}>
      {showContentType && (
        <Stack direction="row" spacing={1} alignItems="center" justifyContent="space-between">
          <Chip size="small" label={getContentTypeLabel(contentType)} />
        </Stack>
      )}
      <Stack data-testid="content-editor" data-code-theme={codeTheme} sx={{ border: 1, borderColor: 'divider', borderRadius: 2, overflow: 'hidden' }}>
        <CodeMirror
          aria-label={ariaLabel}
          value={value}
          minHeight={minHeight}
          maxHeight={maxHeight}
          style={{ maxHeight }}
          basicSetup={{ lineNumbers: true, foldGutter: true, highlightActiveLine: false, highlightActiveLineGutter: false }}
          theme="none"
          editable={!disabled}
          readOnly={disabled}
          extensions={getContentExtensions(contentType, codeTheme, lineWrapping)}
          onChange={onChange}
        />
      </Stack>
    </Stack>
  )
}
