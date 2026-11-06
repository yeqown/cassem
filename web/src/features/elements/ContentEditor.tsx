import CodeMirror from '@uiw/react-codemirror'
import { Chip, FormHelperText, Stack } from '@mui/material'
import { lintGutter } from '@codemirror/lint'
import { useI18n } from '../../i18n'
import { getContentExtensions, getContentTypeLabel } from './contentRegistry'
import { createContentDiagnostics } from './contentDiagnostics'
import { formatValidationMessage, type ContentValidationResult } from './contentValidation'
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
  validation?: ContentValidationResult
  onChange: (value: string) => void
}

export function ContentEditor({ value, contentType, ariaLabel, codeTheme = 'github-light-plus', disabled = false, lineWrapping = true, minRows = 12, maxRows = 24, showContentType = true, validation, onChange }: ContentEditorProps) {
  const { t } = useI18n()
  const minHeight = `${minRows * 24}px`
  const maxHeight = `${maxRows * 24}px`
  const validationMessage = validation && !validation.valid ? formatValidationMessage(validation, t) : ''
  const hasValidationError = Boolean(validationMessage)
  const extensions = [...getContentExtensions(contentType, codeTheme, lineWrapping, t), lintGutter(), ...createContentDiagnostics(contentType, t)]

  return (
    <Stack spacing={1.5}>
      {showContentType && (
        <Stack direction="row" spacing={1} alignItems="center" justifyContent="space-between">
          <Chip size="small" label={getContentTypeLabel(contentType)} />
        </Stack>
      )}
      <Stack data-testid="content-editor" data-code-theme={codeTheme} data-validation-state={hasValidationError ? 'invalid' : 'valid'} sx={{ border: 1, borderColor: hasValidationError ? 'error.main' : 'divider', borderRadius: 2, overflow: 'hidden' }}>
        <CodeMirror
          aria-label={ariaLabel ?? t('elements.rawContent')}
          value={value}
          minHeight={minHeight}
          maxHeight={maxHeight}
          style={{ maxHeight }}
          basicSetup={{ lineNumbers: true, foldGutter: true, highlightActiveLine: false, highlightActiveLineGutter: false }}
          theme="none"
          editable={!disabled}
          readOnly={disabled}
          extensions={extensions}
          onChange={onChange}
        />
      </Stack>
      {hasValidationError && (
        <FormHelperText error data-testid="content-editor-error">
          {validationMessage}
        </FormHelperText>
      )}
    </Stack>
  )
}
