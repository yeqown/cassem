import { useCallback, useEffect, useMemo, useState } from 'react'
import { Box, Stack, Typography } from '@mui/material'
import ReactDiffViewer, { DiffMethod } from 'react-diff-viewer-continued'

const emptyDiffMessage = 'No differences found for this comparison.'

type DiffViewerProps = {
  oldValue: string
  newValue: string
  baseLabel?: string
  compareLabel?: string
  ariaLabel?: string
  contentType?: number | string
}

type HighlightJsApi = {
  highlight: (source: string, options: { language: string; ignoreIllegals: boolean }) => { value: string }
  getLanguage?: (language: string) => unknown
}

const highlightJsVersion = '11.11.1'
const highlightCoreUrl = `https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@${highlightJsVersion}/build/highlight.min.js`
const highlightLanguageUrls: Record<string, string> = {
  json: `https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@${highlightJsVersion}/build/languages/json.min.js`,
  ini: `https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@${highlightJsVersion}/build/languages/ini.min.js`,
}
const loadingScripts = new Map<string, Promise<void>>()

function getHighlightLanguage(contentType?: number | string) {
  const numeric = Number(contentType)
  if (numeric === 1) return 'json'
  if (numeric === 2 || numeric === 3) return 'ini'
  if (typeof contentType !== 'string') return undefined

  const normalized = contentType.toUpperCase()
  if (normalized === 'JSON') return 'json'
  if (normalized === 'TOML' || normalized === 'INI') return 'ini'
  return undefined
}

function loadScript(src: string) {
  const existing = loadingScripts.get(src)
  if (existing) return existing

  const promise = new Promise<void>((resolve, reject) => {
    if (typeof document === 'undefined') {
      resolve()
      return
    }

    const loaded = document.querySelector(`script[src="${src}"][data-loaded="true"]`)
    if (loaded) {
      resolve()
      return
    }

    const script = document.createElement('script')
    script.src = src
    script.async = true
    script.dataset.highlightJsLoader = 'true'
    script.onload = () => {
      script.dataset.loaded = 'true'
      resolve()
    }
    script.onerror = () => reject(new Error(`Failed to load ${src}`))
    document.head.appendChild(script)
  })

  loadingScripts.set(src, promise)
  return promise
}

function ensureHighlight(language: string) {
  const win = window as Window & { hljs?: HighlightJsApi }
  const core = win.hljs ? Promise.resolve() : loadScript(highlightCoreUrl)
  const languageUrl = highlightLanguageUrls[language]

  return core.then(() => {
    const hljs = getHighlightJs()
    if (!languageUrl || hljs?.getLanguage?.(language)) return undefined
    return loadScript(languageUrl)
  })
}

function getHighlightJs() {
  return (window as Window & { hljs?: HighlightJsApi }).hljs
}

function hasHighlightLanguage(language: string) {
  const hljs = getHighlightJs()
  return Boolean(hljs && (!hljs.getLanguage || hljs.getLanguage(language)))
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

export function DiffViewer({ oldValue, newValue, baseLabel = 'Base', compareLabel = 'Compare', ariaLabel = 'Diff', contentType }: DiffViewerProps) {
  const hasDiff = oldValue !== newValue
  const highlightLanguage = useMemo(() => getHighlightLanguage(contentType), [contentType])
  const [loadedHighlightLanguage, setLoadedHighlightLanguage] = useState(() => (highlightLanguage && hasHighlightLanguage(highlightLanguage) ? highlightLanguage : undefined))
  const highlightReady = Boolean(highlightLanguage && (loadedHighlightLanguage === highlightLanguage || hasHighlightLanguage(highlightLanguage)))

  useEffect(() => {
    let mounted = true
    if (!highlightLanguage) {
      return () => {
        mounted = false
      }
    }

    ensureHighlight(highlightLanguage)
      .then(() => {
        if (mounted && hasHighlightLanguage(highlightLanguage)) setLoadedHighlightLanguage(highlightLanguage)
      })
      .catch(() => undefined)

    return () => {
      mounted = false
    }
  }, [highlightLanguage])

  const renderContent = useCallback(
    (source: string) => {
      if (!highlightLanguage || !highlightReady) return <span>{source}</span>

      try {
        const highlighted = getHighlightJs()?.highlight(source, { language: highlightLanguage, ignoreIllegals: true }).value
        if (!highlighted) return <span>{source}</span>
        return <span className="diff-syntax" dangerouslySetInnerHTML={{ __html: highlighted }} />
      } catch {
        return <span>{source}</span>
      }
    },
    [highlightLanguage, highlightReady],
  )

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
      <Box data-testid="diff-scroll-content" sx={{ minWidth: '1000px' }}>
        <Box
          data-testid="diff-header"
          sx={{
            display: 'grid',
            gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 1fr)',
            width: '100%',
            minWidth: '1000px',
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
          <Stack
            sx={{
              '& table': {
                borderCollapse: 'collapse !important',
                width: '100% !important',
                minWidth: '1000px',
              },
              '& td': { borderTop: '0 !important' },
              '& .hljs-attr, & .hljs-attribute': { color: '#c2410c' },
              '& .hljs-string': { color: '#0550ae' },
              '& .hljs-number, & .hljs-literal': { color: '#4f46e5' },
              '& .hljs-comment': { color: '#57606a' },
              '& .hljs-keyword': { color: '#b42318' },
              '& .hljs-section': { color: '#7c3aed' },
            }}
          >
            <ReactDiffViewer
              oldValue={oldValue}
              newValue={newValue}
              splitView
              compareMethod={DiffMethod.WORDS}
              showDiffOnly={false}
              useDarkTheme={false}
              hideLineNumbers={false}
              styles={diffStyles}
              renderContent={renderContent}
            />
          </Stack>
        )}
      </Box>
    </Box>
  )
}
