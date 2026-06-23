import { render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { DiffViewer } from './DiffViewer'

declare global {
  interface Window {
    hljs?: {
      highlight: ReturnType<typeof vi.fn>
      registerLanguage: ReturnType<typeof vi.fn>
    }
  }
}

const originalHljs = window.hljs

function installTestHljs() {
  window.hljs = {
    highlight: vi.fn((source: string) => ({ value: source.replaceAll('title', '<span class="hljs-attr">title</span>') })),
    registerLanguage: vi.fn(),
  }
}

describe('DiffViewer', () => {
  afterEach(() => {
    window.hljs = originalHljs
  })
  it('renders raw old and new values as a split diff without ANSI escape codes', async () => {
    render(<DiffViewer oldValue="timeout=30s" newValue="timeout=45s" baseLabel="Current" compareLabel="Target" />)

    const viewer = screen.getByLabelText('Diff')
    expect(viewer).toHaveAttribute('data-variant', 'split')
    expect(viewer).toHaveAttribute('data-wrap', 'false')
    expect(viewer).not.toHaveTextContent(/\[31m|\[32m|\[0m/)
    expect(screen.getByText('Current')).toBeInTheDocument()
    expect(screen.getByText('Target')).toBeInTheDocument()

    await waitFor(() => {
      expect(viewer).toHaveTextContent('timeout=30s')
      expect(viewer).toHaveTextContent('timeout=45s')
    })
  })

  it('shows empty diff message when raw values match', () => {
    render(<DiffViewer oldValue="same=value" newValue="same=value" />)

    expect(screen.getByLabelText('Diff')).toHaveTextContent('No differences found for this comparison.')
  })

  it('keeps long diff lines unwrapped for horizontal scanning', () => {
    render(<DiffViewer oldValue={`value=${'abc123'.repeat(30)}`} newValue={`value=${'abc124'.repeat(30)}`} />)

    expect(screen.getByLabelText('Diff')).toHaveAttribute('data-wrap', 'false')
  })

  it('keeps the header aligned with the diff table without expanding to long-line width', () => {
    render(<DiffViewer oldValue={`value=${'abc123'.repeat(80)}`} newValue={`value=${'abc124'.repeat(80)}`} />)

    expect(screen.getByTestId('diff-scroll-content')).toHaveStyle({
      minWidth: '1000px',
    })
    expect(screen.getByTestId('diff-scroll-content')).not.toHaveStyle({
      width: 'max-content',
    })
    expect(screen.getByTestId('diff-header')).toHaveStyle({
      minWidth: '1000px',
      width: '100%',
    })
  })

  it('renders highlighted tokens when highlight.js is available for the content type', async () => {
    installTestHljs()

    render(<DiffViewer oldValue={'title = "Old"'} newValue={'title = "New"'} contentType={2} />)

    await waitFor(() => {
      expect(window.hljs?.highlight).toHaveBeenCalledWith(expect.stringContaining('title'), { language: 'ini', ignoreIllegals: true })
    })
    expect(document.querySelector('.hljs-attr')).toHaveTextContent('title')
  })
})
