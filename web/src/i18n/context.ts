import { createContext } from 'react'
import { createLocaleFormatters, type Locale } from './locale'

export type I18nContextValue = ReturnType<typeof createLocaleFormatters> & {
  locale: Locale
  setLocale: (locale: Locale) => void
}

export const I18nContext = createContext<I18nContextValue>({
  locale: 'en',
  setLocale: () => undefined,
  ...createLocaleFormatters('en'),
})
