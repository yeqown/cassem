import { StrictMode } from 'react'
import { render } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

const { showToast } = vi.hoisted(() => ({
  showToast: vi.fn(),
}))

vi.mock('./ToastProvider', () => ({
  useToast: () => ({ showToast }),
}))

import { ErrorState } from './StateView'

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
})
