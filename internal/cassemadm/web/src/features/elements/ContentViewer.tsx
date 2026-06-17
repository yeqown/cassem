import CodeMirror from '@uiw/react-codemirror'
import { Chip, Stack } from '@mui/material'
import { getContentExtensions, getContentTypeLabel } from './contentRegistry'
import type { CodeTheme } from '../../lib/settings'

type ContentViewerProps = {
  value: string
  contentType?: number | string
  ariaLabel?: string
  bordered?: boolean
  codeTheme?: CodeTheme
  lineWrapping?: boolean
  minRows?: number
  showLabel?: boolean
}

export function ContentViewer({ value, contentType, ariaLabel = 'Content preview', bordered = true, codeTheme = 'github-light-plus', lineWrapping = true, minRows = 7, showLabel = true }: ContentViewerProps) {
  const minHeight = `${minRows * 24}px`

  return (
    <Stack spacing={1.5}>
      {showLabel && (
        <Stack direction="row" spacing={1} alignItems="center">
          <Chip size="small" label={getContentTypeLabel(contentType)} />
        </Stack>
      )}
      <Stack
        aria-label={ariaLabel}
        data-code-theme={codeTheme}
        sx={bordered ? { border: 1, borderColor: 'divider', borderRadius: 2, overflow: 'hidden' } : { overflow: 'hidden' }}
      >
        <CodeMirror
          value={value || '-'}
          minHeight={minHeight}
          basicSetup={{ lineNumbers: true, foldGutter: true, highlightActiveLine: false, highlightActiveLineGutter: false }}
          editable={false}
          readOnly
          extensions={getContentExtensions(contentType, codeTheme, lineWrapping)}
        />
      </Stack>
    </Stack>
  )
}
