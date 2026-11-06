import { isLocale, readLocale, type Locale } from './locale'

export function createLocaleStore() {
  let locale = readLocale()
  const listeners = new Set<() => void>()

  return {
    getSnapshot: () => locale,
    subscribe(listener: () => void) {
      listeners.add(listener)
      return () => { listeners.delete(listener) }
    },
    setLocale(next: Locale) {
      if (!isLocale(next) || next === locale) return
      locale = next
      listeners.forEach((listener) => listener())
    },
  }
}
