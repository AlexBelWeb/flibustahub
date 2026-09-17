import { createI18n, type I18n } from 'vue-i18n'
import { fallbackLocale, isLocaleCode, localeLoaders, type LocaleCode } from '@/i18n/registry'

let runtime: I18n | null = null

export function getI18n(): I18n {
  if (!runtime) {
    throw new Error('i18n is not ready')
  }
  return runtime
}

export async function loadLocaleMessages(code: LocaleCode) {
  const loader = localeLoaders[code]
  const mod = await loader()
  return mod.default
}

export async function createAppI18n(initial: string) {
  const locale: LocaleCode = isLocaleCode(initial) ? initial : fallbackLocale
  const [messages, fallbackMessages] = await Promise.all([
    loadLocaleMessages(locale),
    locale === fallbackLocale ? Promise.resolve(null) : loadLocaleMessages(fallbackLocale),
  ])
  runtime = createI18n({
    legacy: false,
    locale,
    fallbackLocale,
    missingWarn: false,
    fallbackWarn: false,
    messages: fallbackMessages
      ? { [locale]: messages, [fallbackLocale]: fallbackMessages }
      : { [locale]: messages },
  })
  return runtime
}

export async function setI18nLocale(code: LocaleCode) {
  const i18n = getI18n()
  if (!i18n.global.availableLocales.includes(code)) {
    i18n.global.setLocaleMessage(code, await loadLocaleMessages(code))
  }
  i18n.global.locale.value = code
  document.documentElement.lang = code
}
