import { StrictMode } from 'react'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'

const { showToast } = vi.hoisted(() => ({
  showToast: vi.fn(),
}))

vi.mock('./ToastProvider', () => ({
  useToast: () => ({ showToast }),
}))

import { ErrorState } from './StateView'
import { useErrorState } from './useErrorState'

afterEach(() => {
  showToast.mockClear()
})

describe('ErrorState', () => {
  it('emits a single toast for the same message under StrictMode remounts', () => {
    render(
      <StrictMode>
        <ErrorState message='page load failed' />
      </StrictMode>,
    )

    expect(showToast).toHaveBeenCalledTimes(1)
    expect(showToast).toHaveBeenCalledWith('page load failed', 'error')
  })

  it('emits the same message again when a new error event is reported', async () => {
    const user = userEvent.setup()

    function ErrorProbe() {
      const [error, reportError, clearError] = useErrorState()
      return (
        <>
          <button onClick={() => reportError('page load failed')}>Report error</button>
          <button onClick={clearError}>Clear error</button>
          {error.message && <ErrorState message={error.message} eventKey={error.eventKey} />}
        </>
      )
    }

    render(<ErrorProbe />)

    await user.click(screen.getByRole('button', { name: /report error/i }))
    await user.click(screen.getByRole('button', { name: /clear error/i }))
    await user.click(screen.getByRole('button', { name: /report error/i }))

    expect(showToast).toHaveBeenCalledTimes(2)
    expect(showToast).toHaveBeenNthCalledWith(1, 'page load failed', 'error')
    expect(showToast).toHaveBeenNthCalledWith(2, 'page load failed', 'error')
  })
})
