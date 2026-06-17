import { render, screen, within } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { DiffViewer } from './DiffViewer'

describe('DiffViewer', () => {
  it('renders ANSI diff as GitHub-like split rows without escape codes', () => {
    const esc = String.fromCharCode(27)

    render(<DiffViewer value={`timeout=${esc}[31m30${esc}[0m${esc}[32m45${esc}[0ms`} baseLabel="Current" compareLabel="Target" />)

    const viewer = screen.getByLabelText('Diff')
    expect(viewer).toHaveAttribute('data-variant', 'split')
    expect(viewer).not.toHaveTextContent(/\[31m|\[32m|\[0m/)
    expect(screen.getByText('Current')).toBeInTheDocument()
    expect(screen.getByText('Target')).toBeInTheDocument()

    const row = screen.getByTestId('diff-row-1')
    expect(row).toHaveAttribute('data-left-tone', 'removed')
    expect(row).toHaveAttribute('data-right-tone', 'added')
    expect(within(row).getByText('-')).toBeInTheDocument()
    expect(within(row).getByText('+')).toBeInTheDocument()
    expect(within(row).getByText('30')).toBeInTheDocument()
    expect(within(row).getByText('45')).toBeInTheDocument()
  })

  it('shows empty diff message when diff text is empty', () => {
    render(<DiffViewer value="" />)

    expect(screen.getByLabelText('Diff')).toHaveTextContent('No differences returned for this comparison.')
  })

  it('keeps long diff lines unwrapped for horizontal scanning', () => {
    render(<DiffViewer value={`value=${'abc123'.repeat(30)}`} />)

    expect(screen.getByLabelText('Diff')).toHaveAttribute('data-wrap', 'false')
  })
})
