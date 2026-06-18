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

  it('uses the selected code theme for the editor background', () => {
    const onChange = vi.fn()

    render(<ContentEditor value={'{"enabled":true}'} contentType={1} disabled={false} codeTheme="one-dark" onChange={onChange} />)

    const container = screen.getByTestId('content-editor')
    const editor = container.querySelector('.cm-editor')
    expect(container).toHaveAttribute('data-code-theme', 'one-dark')
    expect(editor).not.toBeNull()
    expect(getComputedStyle(editor as Element).backgroundColor).toBe('rgb(40, 44, 52)')
  })

  it('can disable line wrapping', () => {
    const onChange = vi.fn()

    render(<ContentEditor value={'value-v1m2p31n123'.repeat(20)} contentType={4} disabled={false} lineWrapping={false} onChange={onChange} />)

    expect(screen.getByTestId('content-editor').querySelector('.cm-lineWrapping')).toBeNull()
  })

  it('does not render formatting controls', () => {
    const onChange = vi.fn()

    render(<ContentEditor value={'{"enabled":true}'} contentType={1} disabled={false} onChange={onChange} />)

    expect(screen.queryByRole('button', { name: /format content/i })).not.toBeInTheDocument()
  })
})
