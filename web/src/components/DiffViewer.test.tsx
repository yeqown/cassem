import { render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { DiffViewer } from './DiffViewer'

describe('DiffViewer', () => {
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
})
