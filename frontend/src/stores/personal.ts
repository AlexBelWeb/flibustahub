import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { getI18n } from '@/i18n'
import { errorMessage } from '@/i18n/errors'
import { parseBackendError } from '@/lib/backend-error'
import { useCatalogStore } from '@/stores/catalog'
import { useToastStore } from '@/stores/toast'
import {
  asImportReport,
  asPersonalSnapshot,
  type PersonalExportResult,
  type PersonalImportReport,
  type PersonalSnapshot,
} from '@/types/personal'

export const usePersonalStore = defineStore('personal', () => {
  const snapshot = ref<PersonalSnapshot>({ unsyncedCount: 0 })
  const snapshotStatus = ref<'idle' | 'loading' | 'ready' | 'error'>('idle')
  const snapshotError = ref('')
  const marksRevision = ref(0)

  const showBadge = computed(
    () => Boolean(snapshot.value.lastExportAt) && snapshot.value.unsyncedCount > 0,
  )

  async function loadSnapshot() {
    snapshotStatus.value = snapshotStatus.value === 'ready' ? 'ready' : 'loading'
    snapshotError.value = ''
    try {
      snapshot.value = asPersonalSnapshot(await window.go.handlers.App.PersonalSnapshot())
      snapshotStatus.value = 'ready'
    } catch (err) {
      const be = parseBackendError(err)
      snapshotError.value = errorMessage(be.code, be.params)
      snapshotStatus.value = snapshotStatus.value === 'ready' ? 'ready' : 'error'
    }
  }

  async function refreshAfterMark() {
    marksRevision.value += 1
    const catalog = useCatalogStore()
    await Promise.all([loadSnapshot(), catalog.loadHome(true)])
  }

  async function setRating(id: number, rating: number) {
    await window.go.handlers.App.SetWorkRating(id, rating)
    const catalog = useCatalogStore()
    catalog.patchWork(id, rating > 0 ? { rating } : { rating: undefined })
    await refreshAfterMark()
  }

  async function setWant(id: number, want: boolean) {
    await window.go.handlers.App.SetWorkWantToRead(id, want)
    const catalog = useCatalogStore()
    catalog.patchWork(id, { wantToRead: want })
    await refreshAfterMark()
  }

  async function setComment(id: number, comment: string) {
    await window.go.handlers.App.SetWorkComment(id, comment)
    await loadSnapshot()
  }

  async function exportTo(path: string): Promise<PersonalExportResult> {
    const result = await window.go.handlers.App.ExportPersonal(path)
    await refreshAfterMark()
    return result
  }

  async function previewImport(path: string): Promise<PersonalImportReport> {
    return asImportReport(await window.go.handlers.App.PreviewPersonalImport(path))
  }

  async function applyImport(path: string): Promise<PersonalImportReport> {
    const report = asImportReport(await window.go.handlers.App.ImportPersonal(path))
    const catalog = useCatalogStore()
    catalog.reset()
    await refreshAfterMark()
    return report
  }

  function reportSaveError(err: unknown, retry: () => void) {
    const toast = useToastStore()
    const be = parseBackendError(err)
    toast.pushError(errorMessage(be.code, be.params), {
      label: String(getI18n().global.t('common.retry')),
      run: retry,
    })
  }

  return {
    snapshot,
    snapshotStatus,
    snapshotError,
    marksRevision,
    showBadge,
    loadSnapshot,
    setRating,
    setWant,
    setComment,
    exportTo,
    previewImport,
    applyImport,
    reportSaveError,
  }
})
