export const codeThemeOptions = [
  { value: 'github-light-plus', label: 'GitHub Light+' },
  { value: 'github-light', label: 'GitHub Light' },
  { value: 'one-dark', label: 'One Dark' },
] as const

export const uiThemeOptions = [
  { value: 'cassem-blue', label: 'Cassem Blue', primary: '#2454ff', secondary: '#00a3a3' },
  { value: 'teal', label: 'Teal', primary: '#00897b', secondary: '#f59e0b' },
  { value: 'purple', label: 'Purple', primary: '#7c3aed', secondary: '#06b6d4' },
  { value: 'orange', label: 'Orange', primary: '#f97316', secondary: '#2563eb' },
] as const

export type CodeTheme = (typeof codeThemeOptions)[number]['value']
export type UITheme = (typeof uiThemeOptions)[number]['value']

export type AppSettings = {
  codeTheme: CodeTheme
  uiTheme: UITheme
  editorLineWrapping: boolean
}

export const defaultSettings: AppSettings = {
  codeTheme: 'github-light-plus',
  uiTheme: 'cassem-blue',
  editorLineWrapping: true,
}

const settingsStorageKey = 'cassem.settings'
export const settingsChangedEvent = 'cassem.settings.changed'

function isCodeTheme(value: unknown): value is CodeTheme {
  return codeThemeOptions.some((option) => option.value === value)
}

function isUITheme(value: unknown): value is UITheme {
  return uiThemeOptions.some((option) => option.value === value)
}

export function readSettings(): AppSettings {
  try {
    const parsed = JSON.parse(localStorage.getItem(settingsStorageKey) || '{}') as Partial<AppSettings>
    return {
      ...defaultSettings,
      codeTheme: isCodeTheme(parsed.codeTheme) ? parsed.codeTheme : defaultSettings.codeTheme,
      uiTheme: isUITheme(parsed.uiTheme) ? parsed.uiTheme : defaultSettings.uiTheme,
      editorLineWrapping: typeof parsed.editorLineWrapping === 'boolean' ? parsed.editorLineWrapping : defaultSettings.editorLineWrapping,
    }
  } catch {
    return defaultSettings
  }
}

export function updateSettings(next: Partial<AppSettings>) {
  let current: Record<string, unknown> = {}
  try {
    current = JSON.parse(localStorage.getItem(settingsStorageKey) || '{}') as Record<string, unknown>
  } catch {
    current = {}
  }

  const merged = { ...current }
  if (next.codeTheme !== undefined) merged.codeTheme = isCodeTheme(next.codeTheme) ? next.codeTheme : defaultSettings.codeTheme
  if (next.uiTheme !== undefined) merged.uiTheme = isUITheme(next.uiTheme) ? next.uiTheme : defaultSettings.uiTheme
  if (next.editorLineWrapping !== undefined) merged.editorLineWrapping = next.editorLineWrapping
  localStorage.setItem(settingsStorageKey, JSON.stringify(merged))
  window.dispatchEvent(new Event(settingsChangedEvent))
}

export function getUIThemeOption(value: UITheme) {
  return uiThemeOptions.find((option) => option.value === value) || uiThemeOptions[0]
}
