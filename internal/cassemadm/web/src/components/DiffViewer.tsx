import { Box, Typography } from '@mui/material'

const esc = String.fromCharCode(27)
const ansiPattern = new RegExp(`${esc}\\[(31|32|0)m`, 'g')
const emptyDiffMessage = 'No differences returned for this comparison.'

type DiffTone = 'added' | 'removed' | 'plain'

type DiffCell = {
  text: string
  tone: DiffTone
}

type DiffRow = {
  left: DiffCell[]
  right: DiffCell[]
  leftNumber: number | null
  rightNumber: number | null
}

type DiffViewerProps = {
  value: string
  baseLabel?: string
  compareLabel?: string
  ariaLabel?: string
}

function parseAnsiChunks(value: string) {
  const chunks: DiffCell[] = []
  let tone: DiffTone = 'plain'
  let cursor = 0

  for (const match of value.matchAll(ansiPattern)) {
    if (match.index > cursor) {
      chunks.push({ text: value.slice(cursor, match.index), tone })
    }
    tone = match[1] === '31' ? 'removed' : match[1] === '32' ? 'added' : 'plain'
    cursor = match.index + match[0].length
  }

  if (cursor < value.length) {
    chunks.push({ text: value.slice(cursor), tone })
  }

  return chunks.filter((chunk) => chunk.text.length > 0)
}

function getCellsText(cells: DiffCell[]) {
  return cells.map((cell) => cell.text).join('')
}

function getCellTone(cells: DiffCell[]): DiffTone {
  if (cells.some((cell) => cell.tone === 'removed')) return 'removed'
  if (cells.some((cell) => cell.tone === 'added')) return 'added'
  return 'plain'
}

function parseDiffRows(value: string) {
  const chunks = parseAnsiChunks(value)
  const rows: DiffRow[] = []
  let left: DiffCell[] = []
  let right: DiffCell[] = []
  let leftNumber = 1
  let rightNumber = 1

  function hasContent() {
    return left.length > 0 || right.length > 0
  }

  function pushRow() {
    if (!hasContent()) return

    const hasLeft = getCellsText(left).length > 0
    const hasRight = getCellsText(right).length > 0
    rows.push({
      left,
      right,
      leftNumber: hasLeft ? leftNumber : null,
      rightNumber: hasRight ? rightNumber : null,
    })
    if (hasLeft) leftNumber += 1
    if (hasRight) rightNumber += 1
    left = []
    right = []
  }

  for (const chunk of chunks) {
    const parts = chunk.text.split('\n')
    parts.forEach((part, index) => {
      if (part) {
        if (chunk.tone !== 'added') left.push({ text: part, tone: chunk.tone })
        if (chunk.tone !== 'removed') right.push({ text: part, tone: chunk.tone })
      }
      if (index < parts.length - 1) pushRow()
    })
  }

  pushRow()
  return rows
}

function renderCells(cells: DiffCell[], side: 'left' | 'right') {
  if (cells.length === 0) return null
  return cells.map((cell, index) => (
    <Box
      key={`${side}-${index}`}
      component="span"
      sx={{
        bgcolor: cell.tone === 'removed' ? 'error.light' : cell.tone === 'added' ? 'success.light' : 'transparent',
        color: 'text.primary',
      }}
    >
      {cell.text}
    </Box>
  ))
}

function toneBackground(tone: DiffTone) {
  if (tone === 'removed') return 'rgba(255, 235, 233, 0.85)'
  if (tone === 'added') return 'rgba(218, 251, 225, 0.85)'
  return 'background.paper'
}

export function DiffViewer({ value, baseLabel = 'Base', compareLabel = 'Compare', ariaLabel = 'Diff' }: DiffViewerProps) {
  const rows = parseDiffRows(value)

  return (
    <Box
      aria-label={ariaLabel}
      data-variant="split"
      data-wrap="false"
      sx={{
        border: 1,
        borderColor: 'divider',
        borderRadius: 0,
        overflowX: 'auto',
        bgcolor: 'background.paper',
      }}
    >
      <Box sx={{ minWidth: 900 }}>
        <Box
          sx={{
            display: 'grid',
            gridTemplateColumns: '56px minmax(360px, 1fr) 56px minmax(360px, 1fr)',
            borderBottom: 1,
            borderColor: 'divider',
            bgcolor: 'info.light',
          }}
        >
          <Box sx={{ px: 1, py: 0.75, borderRight: 1, borderColor: 'divider' }} />
          <Typography variant="caption" fontWeight={700} sx={{ px: 1.5, py: 0.75, borderRight: 1, borderColor: 'divider' }}>{baseLabel}</Typography>
          <Box sx={{ px: 1, py: 0.75, borderRight: 1, borderColor: 'divider' }} />
          <Typography variant="caption" fontWeight={700} sx={{ px: 1.5, py: 0.75 }}>{compareLabel}</Typography>
        </Box>

        {rows.length === 0 ? (
          <Typography color="text.secondary" sx={{ p: 2, fontFamily: 'monospace' }}>{emptyDiffMessage}</Typography>
        ) : rows.map((row, index) => {
          const leftTone = getCellTone(row.left)
          const rightTone = getCellTone(row.right)
          return (
            <Box
              key={`diff-row-${index}`}
              data-testid={`diff-row-${index + 1}`}
              data-left-tone={leftTone}
              data-right-tone={rightTone}
              sx={{
                display: 'grid',
                gridTemplateColumns: '56px minmax(360px, 1fr) 56px minmax(360px, 1fr)',
                borderBottom: index === rows.length - 1 ? 0 : 1,
                borderColor: 'divider',
                fontFamily: 'monospace',
                fontSize: 13,
              }}
            >
              <Box sx={{ px: 1, py: 0.5, textAlign: 'right', color: 'text.secondary', bgcolor: toneBackground(leftTone), borderRight: 1, borderColor: 'divider', userSelect: 'none' }}>{row.leftNumber ?? ''}</Box>
              <Box sx={{ px: 1.5, py: 0.5, whiteSpace: 'pre', bgcolor: toneBackground(leftTone), borderRight: 1, borderColor: 'divider' }}>
                {leftTone === 'removed' ? <Box component="span" sx={{ pr: 1, color: 'error.main' }}>-</Box> : null}
                {renderCells(row.left, 'left')}
              </Box>
              <Box sx={{ px: 1, py: 0.5, textAlign: 'right', color: 'text.secondary', bgcolor: toneBackground(rightTone), borderRight: 1, borderColor: 'divider', userSelect: 'none' }}>{row.rightNumber ?? ''}</Box>
              <Box sx={{ px: 1.5, py: 0.5, whiteSpace: 'pre', bgcolor: toneBackground(rightTone) }}>
                {rightTone === 'added' ? <Box component="span" sx={{ pr: 1, color: 'success.main' }}>+</Box> : null}
                {renderCells(row.right, 'right')}
              </Box>
            </Box>
          )
        })}
      </Box>
    </Box>
  )
}
