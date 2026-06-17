import CodeMirror from '@uiw/react-codemirror'
import { Chip, Stack } from '@mui/material'
import { getContentLanguage, getContentTypeLabel } from './contentRegistry'

type ContentViewerProps = {
  value: string
  contentType?: number | string
  ariaLabel?: string
}

export function ContentViewer({ value, contentType, ariaLabel = 'Content preview' }: ContentViewerProps) {
  return (
    <Stack spacing={1.5}>
      <Stack direction="row" spacing={1} alignItems="center">
        <Chip size="small" label={getContentTypeLabel(contentType)} />
      </Stack>
      <Stack aria-label={ariaLabel} sx={{ border: 1, borderColor: 'divider', borderRadius: 2, overflow: 'hidden' }}>
        <CodeMirror
          value={value || '-'}
          minHeight="160px"
          basicSetup={{ lineNumbers: true, foldGutter: true, highlightActiveLine: false, highlightActiveLineGutter: false }}
          editable={false}
          readOnly
          extensions={getContentLanguage(contentType)}
        />
      </Stack>
    </Stack>
  )
}
