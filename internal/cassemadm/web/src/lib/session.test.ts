import { afterEach, describe, expect, it } from 'vitest'
import { clearSession, getSession, getStoredUser, storeSession } from './session'

afterEach(() => localStorage.clear())

describe('session helpers', () => {
  it('stores and reads session data', () => {
    storeSession('token', { account: 'superadmin@example.com', nickname: 'Super Admin', status: 1 })

    expect(getSession()).toBe('token')
    expect(getStoredUser()).toEqual({ account: 'superadmin@example.com', nickname: 'Super Admin', status: 1 })
  })

  it('removes corrupt user data', () => {
    localStorage.setItem('cassem.user', '{bad')

    expect(getStoredUser()).toBeNull()
    expect(localStorage.getItem('cassem.user')).toBeNull()
  })

  it('clears session data', () => {
    storeSession('token', { account: 'superadmin@example.com' })
    clearSession()

    expect(getSession()).toBe('')
    expect(getStoredUser()).toBeNull()
  })
})
