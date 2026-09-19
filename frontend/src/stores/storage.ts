import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { getI18n } from '@/i18n'
import { Events } from '@/lib/events'
import { eventsOn } from '@/lib/wails-runtime'
import { useAppStore } from '@/stores/app'
import { useCoversStore } from '@/stores/covers'
import { useImportStore } from '@/stores/import'
import { useToastStore } from '@/stores/toast'
import { emptyStorage, type DumpOffer, type StorageSnapshot } from '@/types/storage'

export const useStorageStore = defineStore('storage', () => {
  const snapshot = ref<StorageSnapshot>(emptyStorage())
  const alertDismissed = ref(false)
  const readerPath = ref('')
  let listening = false
  let wasAvailable = false

  const configured = computed(() => snapshot.value.configured)
  const available = computed(() => snapshot.value.available)
  const unreachable = computed(() => snapshot.value.unreachable)
  const dumpOffer = computed<DumpOffer | null>(() => snapshot.value.dumpOffer ?? null)
  const offline = computed(() => configured.value && !available.value)

  function eventRecord(data: unknown): Record<string, unknown> {
    if (Array.isArray(data) && data.length > 0) {
      return eventRecord(data[0])
    }
    if (data && typeof data === 'object') {
      return data as Record<string, unknown>
    }
    return {}
  }

  function asSnapshot(data: unknown): StorageSnapshot {
    const src = eventRecord(data)
    const offerRaw = src.dumpOffer
    let offer: DumpOffer | null = null
    if (offerRaw && typeof offerRaw === 'object') {
      const rec = offerRaw as Record<string, unknown>
      offer = {
        path: typeof rec.path === 'string' ? rec.path : '',
        name: typeof rec.name === 'string' ? rec.name : '',
        fileVersion: typeof rec.fileVersion === 'string' ? rec.fileVersion : undefined,
        catalogVersion: typeof rec.catalogVersion === 'string' ? rec.catalogVersion : undefined,
      }
      if (!offer.path) {
        offer = null
      }
    }
    return {
      configured: Boolean(src.configured),
      available: Boolean(src.available),
      unreachable: Boolean(src.unreachable),
      libraryRoot: typeof src.libraryRoot === 'string' ? src.libraryRoot : '',
      remapped: Boolean(src.remapped),
      dumpOffer: offer,
    }
  }

  function apply(next: StorageSnapshot) {
    const becameAvailable = next.available && !wasAvailable
    snapshot.value = next
    const app = useAppStore()
    if (app.bootstrap && next.libraryRoot && next.libraryRoot !== app.bootstrap.libraryRoot) {
      app.bootstrap = { ...app.bootstrap, libraryRoot: next.libraryRoot, storage: next }
    } else if (app.bootstrap) {
      app.bootstrap = { ...app.bootstrap, storage: next }
    }
    if (next.remapped) {
      useToastStore().pushInfo(String(getI18n().global.t('storage.remapped')))
    }
    if (becameAvailable) {
      alertDismissed.value = false
      useCoversStore().generation += 1
    }
    wasAvailable = next.available
  }

  function listen() {
    if (listening) {
      return
    }
    listening = true
    eventsOn(Events.StorageChanged, (data) => {
      apply(asSnapshot(data))
    })
  }

  async function check(force: boolean) {
    listen()
    const next = await window.go.handlers.App.CheckStorage(force)
    apply(asSnapshot(next))
  }

  function hydrateFromBootstrap() {
    const data = useAppStore().bootstrap?.storage
    if (data) {
      apply(data)
    }
  }

  async function loadReaderPath() {
    try {
      readerPath.value = (await window.go.handlers.App.ReaderPath()) ?? ''
    } catch {
      readerPath.value = ''
    }
  }

  async function dismissDump() {
    await window.go.handlers.App.DismissDumpOffer()
    snapshot.value = { ...snapshot.value, dumpOffer: null }
  }

  async function chooseFolder(title: string) {
    await useImportStore().chooseFolder(title)
    await check(true)
  }

  return {
    snapshot,
    alertDismissed,
    readerPath,
    configured,
    available,
    unreachable,
    dumpOffer,
    offline,
    listen,
    check,
    hydrateFromBootstrap,
    loadReaderPath,
    dismissDump,
    chooseFolder,
  }
})
