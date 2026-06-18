import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { ContentViewer } from './ContentViewer'
import { getContentLanguage } from './contentRegistry'


describe('ContentViewer', () => {
  it('renders read-only syntax viewer with line numbers and hides content type label by default', () => {
    render(<ContentViewer value={'server.port = 8080'} contentType={2} ariaLabel="Version content" />)

    expect(screen.queryByText('TOML')).not.toBeInTheDocument()
    const viewer = screen.getByLabelText('Version content')
    expect(viewer).toHaveTextContent('server.port = 8080')
    expect(viewer).toHaveAttribute('data-code-theme', 'github-light-plus')
    expect(viewer.querySelector('.cm-lineNumbers')).not.toBeNull()
  })

  it('shows content type label when enabled', () => {
    render(<ContentViewer value={'server.port = 8080'} contentType={2} ariaLabel="Labeled content" showLabel />)

    expect(screen.getByText('TOML')).toBeInTheDocument()
  })

  it('wraps long content lines instead of horizontal scrolling', () => {
    render(<ContentViewer value={'value-v1m2p31n123'.repeat(20)} contentType={4} ariaLabel="Wrapped content" />)

    expect(screen.getByLabelText('Wrapped content').querySelector('.cm-lineWrapping')).not.toBeNull()
  })

  it('can disable line wrapping', () => {
    render(<ContentViewer value={'value-v1m2p31n123'.repeat(20)} contentType={4} ariaLabel="Unwrapped content" lineWrapping={false} />)

    expect(screen.getByLabelText('Unwrapped content').querySelector('.cm-lineWrapping')).toBeNull()
  })

  it('uses the selected code theme for the editor background', () => {
    render(<ContentViewer value={'{"enabled":true}'} contentType={1} ariaLabel="Themed content" codeTheme="one-dark" />)

    const viewer = screen.getByLabelText('Themed content')
    const editor = viewer.querySelector('.cm-editor')
    expect(viewer).toHaveAttribute('data-code-theme', 'one-dark')
    expect(editor).not.toBeNull()
    expect(getComputedStyle(editor as Element).backgroundColor).toBe('rgb(40, 44, 52)')
  })

  it('supports protojson content type enum names without showing the label by default', () => {
    render(<ContentViewer value={'{"enabled":true}'} contentType="JSON" ariaLabel="Enum content" />)

    expect(screen.queryByText('JSON')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Enum content')).toHaveTextContent('"enabled"')
    expect(getContentLanguage('JSON')).toHaveLength(1)
  })

  it('renders placeholder for empty content', () => {
    render(<ContentViewer value="" contentType={4} ariaLabel="Empty content" />)

    expect(screen.getByLabelText('Empty content')).toHaveTextContent('-')
  })
})
