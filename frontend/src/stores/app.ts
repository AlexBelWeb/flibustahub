import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { errorMessage } from '@/i18n/errors'
import { setI18nLocale } from '@/i18n'
import { isLocaleCode, type LocaleCode } from '@/i18n/registry'
import { parseBackendError } from '@/lib/backend-error'
import { Events } from '@/lib/events'
import { eventsOn } from '@/lib/wails-runtime'
import { useToastStore } from '@/stores/toast'
import { useStorageStore } from '@/stores/storage'
import type { Bootstrap, StartupError } from '@/types/bootstrap'
import type { CatalogView } from '@/types/catalog'

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
  const startupError = computed<StartupError | null>(() => bootstrap.value?.startupError ?? null)
  const databaseUpdating = computed(() => bootstrap.value?.databaseUpdating ?? false)
  const catalogOpening = computed(() => bootstrap.value?.catalogOpening ?? false)
  const catalogReady = computed(() => bootstrap.value?.catalogReady ?? false)
  const searchIndexReady = computed(() => bootstrap.value?.searchIndexReady ?? false)
  const sidebarCollapsed = computed(() => bootstrap.value?.sidebarCollapsed ?? false)
  const catalogView = computed<CatalogView>(() =>
    bootstrap.value?.catalogView === 'tile' ? 'tile' : 'table',
  )
  const narrow = ref(false)

  let listeningDB = false
  let listeningIndex = false
  let pollTimer = 0

  function eventRecord(data: unknown): Record<string, unknown> {
    if (Array.isArray(data) && data.length > 0) {
      return eventRecord(data[0])
    }
    if (data && typeof data === 'object') {
      return data as Record<string, unknown>
    }
    return {}
  }

  function applyDBUpdated(data: unknown) {
    const src = eventRecord(data)
    const raw = src.error
    if (raw && typeof raw === 'object' && bootstrap.value) {
      const rec = raw as Record<string, unknown>
      const code = typeof rec.code === 'string' ? rec.code : ''
      if (code) {
        const params =
          rec.params && typeof rec.params === 'object' && !Array.isArray(rec.params)
            ? (rec.params as Record<string, string>)
            : undefined
        bootstrap.value = {
          ...bootstrap.value,
          databaseUpdating: false,
          catalogOpening: false,
          catalogReady: false,
          startupError: { code, params },
        }
      }
    }
    void refresh().then(() => {
      schedulePoll()
    })
  }

  function shouldPoll(): boolean {
    const data = bootstrap.value
    return Boolean(data && !data.startupError && (data.databaseUpdating || data.catalogOpening))
  }

  function schedulePoll() {
    window.clearTimeout(pollTimer)
    pollTimer = 0
    if (!shouldPoll()) {
      return
    }
    pollTimer = window.setTimeout(() => {
      void refresh().then(() => {
        schedulePoll()
      })
    }, 400)
  }

  function listenDBUpdated() {
    if (listeningDB) {
      return
    }
    listeningDB = true
    eventsOn(Events.DBUpdated, applyDBUpdated)
  }

  function listenIndexReady() {
    if (listeningIndex) {
      return
    }
    listeningIndex = true
    eventsOn(Events.SearchIndexReady, () => {
      void refresh()
    })
  }

  function watchViewport() {
    const mq = window.matchMedia('(max-width: 1099px)')
    const apply = () => {
      narrow.value = mq.matches
    }
    apply()
    mq.addEventListener('change', apply)
  }

  function applyDocumentTheme(value: Theme) {
    const root = document.documentElement
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
    const dark = value === 'dark' || (value === 'system' && prefersDark)
    root.classList.toggle('dark', dark)
    root.classList.toggle('light', !dark)
    root.classList.toggle('effects-reduced', reducedEffects.value)
  }

  function applyBootstrap(data: Bootstrap) {
    bootstrap.value = data
    const next: LocaleCode = isLocaleCode(data.locale) ? data.locale : 'en'
    void setI18nLocale(next)
    applyDocumentTheme(data.theme as Theme)
    if (data.storage) {
      useStorageStore().hydrateFromBootstrap()
    }
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

  async function refresh() {
    const data = await window.go.handlers.App.Bootstrap()
    applyBootstrap(data)
  }

  async function load() {
    loading.value = true
    loadError.value = false
    listenDBUpdated()
    listenIndexReady()
    watchViewport()
    try {
      const data = await window.go.handlers.App.Bootstrap()
      applyBootstrap(data)
      schedulePoll()
    } catch {
      loadError.value = true
    } finally {
      loading.value = false
    }
  }

  async function retryStartup() {
    const data = await window.go.handlers.App.RetryStartup()
    applyBootstrap(data)
    schedulePoll()
  }

  async function openLogsDir() {
    await wrap(() => window.go.handlers.App.OpenLogsDir())
  }

  async function openDataDir() {
    await wrap(() => window.go.handlers.App.OpenDataDir())
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
    applyBootstrap(data)
  }

  async function toggleSidebar() {
    await setSidebarCollapsed(!sidebarCollapsed.value)
  }

  async function setSidebarCollapsed(collapsed: boolean) {
    await wrap(() => window.go.handlers.App.SetSidebarCollapsed(collapsed))
    if (bootstrap.value) {
      bootstrap.value = { ...bootstrap.value, sidebarCollapsed: collapsed }
    }
  }

  async function changeCatalogView(view: CatalogView) {
    await wrap(() => window.go.handlers.App.SetCatalogView(view))
    if (bootstrap.value) {
      bootstrap.value = { ...bootstrap.value, catalogView: view }
    }
  }

  const sidebarIconsOnly = computed(() => sidebarCollapsed.value || narrow.value)

  return {
    bootstrap,
    loading,
    loadError,
    locale,
    theme,
    reducedEffects,
    startupError,
    databaseUpdating,
    catalogOpening,
    catalogReady,
    searchIndexReady,
    sidebarCollapsed,
    sidebarIconsOnly,
    catalogView,
    narrow,
    refresh,
    load,
    retryStartup,
    openLogsDir,
    openDataDir,
    changeLocale,
    changeTheme,
    changeEffects,
    toggleSidebar,
    setSidebarCollapsed,
    changeCatalogView,
    applyDocumentTheme,
  }
})
