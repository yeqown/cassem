import { afterEach, describe, expect, it } from 'vitest'
import { defaultSettings, readSettings, updateSettings } from './settings'

describe('settings storage', () => {
  afterEach(() => {
    localStorage.clear()
  })

  it('returns defaults when local storage is empty or invalid', () => {
    expect(readSettings()).toEqual(defaultSettings)

    localStorage.setItem('cassem.settings', '{bad json')

    expect(readSettings()).toEqual(defaultSettings)
  })

  it('updates one setting while preserving existing and unknown fields', () => {
    localStorage.setItem('cassem.settings', JSON.stringify({ codeTheme: 'github-light', uiTheme: 'teal', editorLineWrapping: false, futureFlag: true }))

    updateSettings({ codeTheme: 'one-dark' })

    expect(JSON.parse(localStorage.getItem('cassem.settings') || '{}')).toEqual({
      codeTheme: 'one-dark',
      uiTheme: 'teal',
      editorLineWrapping: false,
      futureFlag: true,
    })
  })

  it('updates UI theme and editor line wrapping settings', () => {
    updateSettings({ uiTheme: 'purple', editorLineWrapping: false })

    expect(readSettings()).toEqual({
      codeTheme: 'github-light-plus',
      uiTheme: 'purple',
      editorLineWrapping: false,
    })
  })
})
