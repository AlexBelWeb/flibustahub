import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { errorMessage } from '@/i18n/errors'
import { setI18nLocale } from '@/i18n'
import { isLocaleCode, type LocaleCode } from '@/i18n/registry'
import { parseBackendError } from '@/lib/backend-error'
import { useToastStore } from '@/stores/toast'
import type { Bootstrap } from '@/types/bootstrap'

export type Theme = 'system' | 'dark' | 'light'
export type Effects = 'auto' | 'full' | 'reduced'

export const useAppStore = defineStore('app', () => {
  const bootstrap = ref<Bootstrap | null>(null)
  const loading = ref(true)
  const loadError = ref(false)

  const locale = computed(() => bootstrap.value?.locale ?? 'en')
  const theme = computed(() => (bootstrap.value?.theme ?? 'system') as Theme)
  const reducedEffects = computed(
    () => bootstrap.value?.capabilities.effectiveEffects === 'reduced',
  )

  function applyDocumentTheme(value: Theme) {
    const root = document.documentElement
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
    const dark = value === 'dark' || (value === 'system' && prefersDark)
    root.classList.toggle('dark', dark)
    root.classList.toggle('light', !dark)
    root.classList.toggle('effects-reduced', reducedEffects.value)
  }

  async function wrap<T>(fn: () => Promise<T>): Promise<T> {
    try {
      return await fn()
    } catch (err) {
      const parsed = parseBackendError(err)
      useToastStore().pushError(errorMessage(parsed.code, parsed.params))
      throw err
    }
  }

  async function load() {
    loading.value = true
    loadError.value = false
    try {
      const data = await wrap(() => window.go.handlers.App.Bootstrap())
      bootstrap.value = data
      const next: LocaleCode = isLocaleCode(data.locale) ? data.locale : 'en'
      await setI18nLocale(next)
      applyDocumentTheme(data.theme as Theme)
    } catch {
      loadError.value = true
    } finally {
      loading.value = false
    }
  }

  async function changeLocale(code: LocaleCode) {
    await wrap(() => window.go.handlers.App.SetLocale(code))
    await setI18nLocale(code)
    if (bootstrap.value) {
      bootstrap.value = { ...bootstrap.value, locale: code }
    }
  }

  async function changeTheme(value: Theme) {
    await wrap(() => window.go.handlers.App.SetTheme(value))
    if (bootstrap.value) {
      bootstrap.value = { ...bootstrap.value, theme: value }
    }
    applyDocumentTheme(value)
  }

  async function changeEffects(value: Effects) {
    await wrap(() => window.go.handlers.App.SetVisualEffects(value))
    const data = await wrap(() => window.go.handlers.App.Bootstrap())
    bootstrap.value = data
    applyDocumentTheme(data.theme as Theme)
  }

  return {
    bootstrap,
    loading,
    loadError,
    locale,
    theme,
    reducedEffects,
    load,
    changeLocale,
    changeTheme,
    changeEffects,
    applyDocumentTheme,
  }
})
