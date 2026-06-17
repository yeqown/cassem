import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ContentEditor } from './ContentEditor'


describe('ContentEditor', () => {
  it('renders syntax editor with line numbers and content type label', () => {
    const onChange = vi.fn()

    render(<ContentEditor value={'{"enabled":true}'} contentType={1} disabled={false} onChange={onChange} />)

    expect(screen.getByText('JSON')).toBeInTheDocument()
    const editor = screen.getByTestId('content-editor')
    expect(editor).toHaveTextContent('"enabled"')
    expect(editor.querySelector('.cm-lineNumbers')).not.toBeNull()
  })

  it('does not render formatting controls', () => {
    const onChange = vi.fn()

    render(<ContentEditor value={'{"enabled":true}'} contentType={1} disabled={false} onChange={onChange} />)

    expect(screen.queryByRole('button', { name: /format content/i })).not.toBeInTheDocument()
  })
})
