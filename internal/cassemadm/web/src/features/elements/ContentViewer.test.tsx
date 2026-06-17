import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { ContentViewer } from './ContentViewer'
import { getContentLanguage } from './contentRegistry'


describe('ContentViewer', () => {
  it('renders read-only syntax viewer with content type label and line numbers', () => {
    render(<ContentViewer value={'server.port = 8080'} contentType={2} ariaLabel="Version content" />)

    expect(screen.getByText('TOML')).toBeInTheDocument()
    const viewer = screen.getByLabelText('Version content')
    expect(viewer).toHaveTextContent('server.port = 8080')
    expect(viewer).toHaveAttribute('data-code-theme', 'github-light-plus')
    expect(viewer.querySelector('.cm-lineNumbers')).not.toBeNull()
  })

  it('wraps long content lines instead of horizontal scrolling', () => {
    render(<ContentViewer value={'value-v1m2p31n123'.repeat(20)} contentType={4} ariaLabel="Wrapped content" />)

    expect(screen.getByLabelText('Wrapped content').querySelector('.cm-lineWrapping')).not.toBeNull()
  })

  it('can disable line wrapping', () => {
    render(<ContentViewer value={'value-v1m2p31n123'.repeat(20)} contentType={4} ariaLabel="Unwrapped content" lineWrapping={false} />)

    expect(screen.getByLabelText('Unwrapped content').querySelector('.cm-lineWrapping')).toBeNull()
  })

  it('uses the selected code theme', () => {
    render(<ContentViewer value={'{"enabled":true}'} contentType={1} ariaLabel="Themed content" codeTheme="one-dark" />)

    expect(screen.getByLabelText('Themed content')).toHaveAttribute('data-code-theme', 'one-dark')
  })

  it('supports protojson content type enum names', () => {
    render(<ContentViewer value={'{"enabled":true}'} contentType="JSON" ariaLabel="Enum content" />)

    expect(screen.getByText('JSON')).toBeInTheDocument()
    expect(screen.getByLabelText('Enum content')).toHaveTextContent('"enabled"')
    expect(getContentLanguage('JSON')).toHaveLength(1)
  })

  it('renders placeholder for empty content', () => {
    render(<ContentViewer value="" contentType={4} ariaLabel="Empty content" />)

    expect(screen.getByLabelText('Empty content')).toHaveTextContent('-')
  })
})
