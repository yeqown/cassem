import { en, zhCN, type MessageKey } from './messages'

export type Locale = 'en' | 'zh-CN'
export type TranslationParams = Record<string, string | number>
export type Translator = (key: MessageKey, params?: TranslationParams) => string
export const localeStorageKey = 'cassem.locale'
export const languageOptions = [
  { value: 'en', label: 'English' },
  { value: 'zh-CN', label: '简体中文' },
] as const

export function isLocale(value: unknown): value is Locale {
  return value === 'en' || value === 'zh-CN'
}

export function browserLocale(): Locale {
  if (typeof navigator === 'undefined') return 'en'
  const languages = navigator.languages?.length ? navigator.languages : [navigator.language]
  for (const language of languages) {
    if (/^zh(?:-|$)/i.test(language)) return 'zh-CN'
    if (/^en(?:-|$)/i.test(language)) return 'en'
  }
  return 'en'
}

export function readLocale(): Locale {
  try {
    const saved = localStorage.getItem(localeStorageKey)
    if (isLocale(saved)) return saved
  } catch {
    // The interface remains usable when browser storage is unavailable.
  }
  return browserLocale()
}

export function translate(locale: Locale, key: MessageKey, params: TranslationParams = {}): string {
  const message = locale === 'zh-CN' ? zhCN[key] : en[key]
  return (message ?? en[key] ?? key).replace(/\{(\w+)\}/g, (placeholder, name: string) =>
    Object.hasOwn(params, name) ? String(params[name]) : placeholder,
  )
}

export function createLocaleFormatters(locale: Locale) {
  return {
    t: ((key, params) => translate(locale, key, params)) as Translator,
    formatDateTime(value: Date | number | string, options?: Intl.DateTimeFormatOptions) {
      const date = value instanceof Date ? value : new Date(value)
      if (Number.isNaN(date.getTime())) return '-'
      return date.toLocaleString(locale, options)
    },
    formatNumber(value: number, options?: Intl.NumberFormatOptions) {
      return new Intl.NumberFormat(locale, options).format(value)
    },
  }
}
