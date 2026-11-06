import { useCallback, useEffect, useMemo, useState, useSyncExternalStore, type ReactNode } from 'react'
import { I18nContext } from './context'
import { browserLocale, createLocaleFormatters, isLocale, localeStorageKey, translate, type Locale, type Translator } from './locale'
import { createLocaleStore } from './store'

export function I18nProvider({ children }: { children: ReactNode }) {
  const [store] = useState(createLocaleStore)
  const locale = useSyncExternalStore(store.subscribe, store.getSnapshot, store.getSnapshot)
  const setLocale = useCallback((next: Locale) => {
    if (!isLocale(next)) return
    store.setLocale(next)
    try {
      localStorage.setItem(localeStorageKey, next)
    } catch {
      // Switching still works for this visit if persistence is blocked.
    }
  }, [store])
  const value = useMemo(() => ({
    locale,
    setLocale,
    ...createLocaleFormatters(locale),
    // Pending requests may retain an earlier callback; resolve their feedback
    // against the latest preference without restarting the request or form.
    t: ((key, params) => translate(store.getSnapshot(), key, params)) as Translator,
  }), [locale, setLocale, store])

  useEffect(() => {
    document.documentElement.lang = locale
    document.title = value.t('common.pageTitle')
  }, [locale, value])

  useEffect(() => {
    function syncLocale(event: StorageEvent) {
      if (event.key !== localeStorageKey && event.key !== null) return
      store.setLocale(isLocale(event.newValue) ? event.newValue : browserLocale())
    }
    window.addEventListener('storage', syncLocale)
    return () => window.removeEventListener('storage', syncLocale)
  }, [store])

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}
