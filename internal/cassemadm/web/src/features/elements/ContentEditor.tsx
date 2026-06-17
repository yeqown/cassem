import CodeMirror from '@uiw/react-codemirror'
import { Chip, Stack } from '@mui/material'
import { getContentLanguage, getContentTypeLabel } from './contentRegistry'

type ContentEditorProps = {
  value: string
  contentType?: number | string
  disabled?: boolean
  minRows?: number
  onChange: (value: string) => void
}

export function ContentEditor({ value, contentType, disabled = false, minRows = 12, onChange }: ContentEditorProps) {
  const minHeight = `${minRows * 24}px`

  return (
    <Stack spacing={1.5}>
      <Stack direction="row" spacing={1} alignItems="center" justifyContent="space-between">
        <Chip size="small" label={getContentTypeLabel(contentType)} />
      </Stack>
      <Stack data-testid="content-editor" sx={{ border: 1, borderColor: 'divider', borderRadius: 2, overflow: 'hidden' }}>
        <CodeMirror
          aria-label="Raw content"
          value={value}
          minHeight={minHeight}
          basicSetup={{ lineNumbers: true, foldGutter: true }}
          editable={!disabled}
          readOnly={disabled}
          extensions={getContentLanguage(contentType)}
          onChange={onChange}
        />
      </Stack>
    </Stack>
  )
}
