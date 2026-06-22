import { render, screen, waitFor } from '@testing-library/react'
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

  it('can hide the content type label', () => {
    const onChange = vi.fn()

    render(<ContentEditor value={'{"enabled":true}'} contentType={1} disabled={false} showContentType={false} onChange={onChange} />)

    expect(screen.queryByText('JSON')).not.toBeInTheDocument()
  })

  it('limits editor height after max rows', () => {
    const onChange = vi.fn()

    render(<ContentEditor value={'line\n'.repeat(30)} contentType={4} disabled={false} minRows={4} maxRows={8} onChange={onChange} />)

    const editorShell = screen.getByTestId('content-editor').querySelector('.cm-theme-none')
    expect(editorShell).not.toBeNull()
    expect((editorShell as HTMLElement).style.maxHeight).toBe('192px')
  })

  it('does not highlight an active line over selected content', () => {
    const onChange = vi.fn()

    render(<ContentEditor value={'first\nlast'} contentType={4} disabled={false} onChange={onChange} />)

    expect(screen.getByTestId('content-editor').querySelector('.cm-activeLine')).toBeNull()
  })

  it('does not render formatting controls', () => {
    const onChange = vi.fn()

    render(<ContentEditor value={'{"enabled":true}'} contentType={1} disabled={false} onChange={onChange} />)

    expect(screen.queryByRole('button', { name: /format content/i })).not.toBeInTheDocument()
  })

  it('renders validation error text and lint gutter', async () => {
    const onChange = vi.fn()

    render(<ContentEditor value={'{"enabled":'} contentType={1} disabled={false} validation={{ valid: false, line: 1, column: 12, message: 'Unexpected end of JSON input' }} onChange={onChange} />)

    const editor = screen.getByTestId('content-editor')
    expect(editor).toHaveAttribute('data-validation-state', 'invalid')
    await waitFor(() => expect(editor.querySelector('.cm-lint-marker')).not.toBeNull())
    expect(screen.getByTestId('content-editor-error')).toHaveTextContent('Line 1, Column 12: Unexpected end of JSON input')
  })

  it('does not render validation error text for valid content', () => {
    const onChange = vi.fn()

    render(<ContentEditor value={'{"enabled":true}'} contentType={1} disabled={false} validation={{ valid: true }} onChange={onChange} />)

    expect(screen.getByTestId('content-editor')).toHaveAttribute('data-validation-state', 'valid')
    expect(screen.queryByTestId('content-editor-error')).not.toBeInTheDocument()
  })
})
