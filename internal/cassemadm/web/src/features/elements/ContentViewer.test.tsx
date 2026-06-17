import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { ContentViewer } from './ContentViewer'


describe('ContentViewer', () => {
  it('renders read-only syntax viewer with content type label and line numbers', () => {
    render(<ContentViewer value={'server.port = 8080'} contentType={2} ariaLabel="Version content" />)

    expect(screen.getByText('TOML')).toBeInTheDocument()
    const viewer = screen.getByLabelText('Version content')
    expect(viewer).toHaveTextContent('server.port = 8080')
    expect(viewer.querySelector('.cm-lineNumbers')).not.toBeNull()
  })

  it('renders placeholder for empty content', () => {
    render(<ContentViewer value="" contentType={4} ariaLabel="Empty content" />)

    expect(screen.getByLabelText('Empty content')).toHaveTextContent('-')
  })
})
