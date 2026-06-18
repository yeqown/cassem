import { Box, Stack, Typography } from '@mui/material'
import ReactDiffViewer, { DiffMethod } from 'react-diff-viewer-continued'

const emptyDiffMessage = 'No differences found for this comparison.'

type DiffViewerProps = {
  oldValue: string
  newValue: string
  baseLabel?: string
  compareLabel?: string
  ariaLabel?: string
}

const diffStyles = {
  variables: {
    light: {
      diffViewerBackground: '#ffffff',
      diffViewerColor: '#24292f',
      addedBackground: '#dafbe1',
      addedColor: '#24292f',
      removedBackground: '#ffebe9',
      removedColor: '#24292f',
      wordAddedBackground: '#aceebb',
      wordRemovedBackground: '#ffd7d5',
      addedGutterBackground: '#aceebb',
      removedGutterBackground: '#ffd7d5',
      gutterBackground: '#f6f8fa',
      gutterBackgroundDark: '#f6f8fa',
      highlightBackground: '#fff8c5',
      highlightGutterBackground: '#fff8c5',
      codeFoldGutterBackground: '#f6f8fa',
      codeFoldBackground: '#f6f8fa',
      emptyLineBackground: '#ffffff',
      gutterColor: '#57606a',
      addedGutterColor: '#24292f',
      removedGutterColor: '#24292f',
      codeFoldContentColor: '#57606a',
    },
  },
  diffContainer: {
    borderRadius: 0,
    fontFamily: 'ui-monospace, SFMono-Regular, SFMono, Consolas, Liberation Mono, Menlo, monospace',
    fontSize: '13px',
  },
  line: {
    padding: '0 10px',
    minHeight: '22px',
    lineHeight: '22px',
    border: 0,
  },
  gutter: {
    padding: '0 10px',
    minWidth: '48px',
    border: 0,
    userSelect: 'none',
  },
  marker: {
    padding: '0 4px',
    userSelect: 'none',
  },
  contentText: {
    whiteSpace: 'pre',
  },
} as const

export function DiffViewer({ oldValue, newValue, baseLabel = 'Base', compareLabel = 'Compare', ariaLabel = 'Diff' }: DiffViewerProps) {
  const hasDiff = oldValue !== newValue

  return (
    <Box
      aria-label={ariaLabel}
      data-variant="split"
      data-wrap="false"
      sx={{
        border: '1px solid #d0d7de',
        borderRadius: 0,
        overflowX: 'auto',
        bgcolor: '#ffffff',
      }}
    >
      <Box sx={{ minWidth: 900 }}>
        <Box
          sx={{
            display: 'grid',
            gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 1fr)',
            borderBottom: '1px solid #d0d7de',
            bgcolor: '#f6f8fa',
            color: '#24292f',
          }}
        >
          <Typography variant="caption" fontWeight={700} sx={{ px: 1.5, py: 0.75, borderRight: '1px solid #d0d7de' }}>
            {baseLabel}
          </Typography>
          <Typography variant="caption" fontWeight={700} sx={{ px: 1.5, py: 0.75 }}>
            {compareLabel}
          </Typography>
        </Box>

        {!hasDiff ? (
          <Typography color="text.secondary" sx={{ p: 2, fontFamily: 'ui-monospace, SFMono-Regular, SFMono, Consolas, Liberation Mono, Menlo, monospace' }}>
            {emptyDiffMessage}
          </Typography>
        ) : (
          <Stack sx={{ '& table': { borderCollapse: 'collapse !important' }, '& td': { borderTop: '0 !important' } }}>
            <ReactDiffViewer
              oldValue={oldValue}
              newValue={newValue}
              splitView
              compareMethod={DiffMethod.WORDS}
              showDiffOnly={false}
              useDarkTheme={false}
              hideLineNumbers={false}
              styles={diffStyles}
            />
          </Stack>
        )}
      </Box>
    </Box>
  )
}
