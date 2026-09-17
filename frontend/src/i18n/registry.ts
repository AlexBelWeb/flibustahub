export const locales = [
  { code: 'ru', nativeName: 'Русский', dir: 'ltr' },
  { code: 'en', nativeName: 'English', dir: 'ltr' },
] as const

export type LocaleCode = (typeof locales)[number]['code']

export const fallbackLocale: LocaleCode = 'en'

export const localeLoaders: Record<
  LocaleCode,
  () => Promise<{ default: Record<string, unknown> }>
> = {
  ru: () => import('@/locales/ru.json'),
  en: () => import('@/locales/en.json'),
}

export function isLocaleCode(value: string): value is LocaleCode {
  return locales.some((item) => item.code === value)
}
