import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { errorMessage } from '@/i18n/errors'
import { Events } from '@/lib/events'
import { parseBackendError, type BackendError } from '@/lib/backend-error'
import { eventsOn, quitApp } from '@/lib/wails-runtime'
import { useAppStore } from '@/stores/app'
import { useCatalogStore } from '@/stores/catalog'
import type { ImportPreview, ImportProgress, ImportReport } from '@/types/import'
import { isImportReport } from '@/types/import'

function asRecord(data: unknown): Record<string, unknown> {
  if (Array.isArray(data) && data.length > 0) {
    return asRecord(data[0])
  }
  if (data && typeof data === 'object') {
    return data as Record<string, unknown>
  }
  return {}
}

function parseProgress(data: unknown): ImportProgress {
  const src = asRecord(data)
  return {
    phase: typeof src.phase === 'string' ? src.phase : 'records',
    recordsSeen: Number(src.recordsSeen) || 0,
    bytesDone: Number(src.bytesDone) || 0,
    bytesTotal: Number(src.bytesTotal) || 0,
    committed: Boolean(src.committed),
  }
}

export const useImportStore = defineStore('import', () => {
  const cardLoading = ref(true)
  const preview = ref<ImportPreview | null>(null)
  const previewError = ref<BackendError | null>(null)
  const lastReport = ref<ImportReport | null>(null)

  const modalOpen = ref(false)
  const modalStage = ref<'progress' | 'report'>('progress')
  const running = ref(false)
  const progress = ref<ImportProgress | null>(null)
  const progressStartedAt = ref(0)
  const result = ref<ImportReport | null>(null)
  const runError = ref<BackendError | null>(null)

  const confirmReimport = ref(false)
  const confirmCancel = ref(false)
  const confirmClose = ref(false)
  const closeCommitted = ref(false)
  const closingApp = ref(false)

  const libraryRoot = computed(() => useAppStore().bootstrap?.libraryRoot ?? '')
  const hasFolder = computed(() => libraryRoot.value.trim() !== '')
  const visiblePhase = computed(() => {
    const phase = progress.value?.phase
    if (!phase || phase === 'reading') {
      return 'records'
    }
    return phase
  })
  const canCancel = computed(() => running.value && !progress.value?.committed)

  let stopProgress: (() => void) | null = null
  let stopClose: (() => void) | null = null

  function listen() {
    if (!stopProgress) {
      stopProgress = eventsOn(Events.ImportProgress, (data) => {
        progress.value = parseProgress(data)
      })
    }
    if (!stopClose) {
      stopClose = eventsOn(Events.ImportCloseRequested, (data) => {
        const src = asRecord(data)
        closeCommitted.value = Boolean(src.committed)
        confirmClose.value = true
      })
    }
  }

  async function loadLastReport() {
    try {
      const report = await window.go.handlers.App.LastImportReport()
      lastReport.value = isImportReport(report) ? report : null
    } catch {
      lastReport.value = null
    }
  }

  async function loadCard() {
    if (!useAppStore().catalogReady) {
      cardLoading.value = false
      return
    }
    cardLoading.value = true
    previewError.value = null
    try {
      const report = await window.go.handlers.App.LastImportReport()
      lastReport.value = isImportReport(report) ? report : null
      if (!hasFolder.value) {
        preview.value = null
        return
      }
      try {
        preview.value = await window.go.handlers.App.PreviewImport()
        previewError.value = null
      } catch (err) {
        preview.value = null
        previewError.value = parseBackendError(err)
      }
    } catch (err) {
      lastReport.value = null
      previewError.value = parseBackendError(err)
    } finally {
      cardLoading.value = false
    }
  }

  async function chooseFolder(dialogTitle: string) {
    try {
      const path = await window.go.handlers.App.SelectLibraryRoot(dialogTitle)
      if (!path) {
        return
      }
      const app = useAppStore()
      if (app.bootstrap) {
        app.bootstrap = { ...app.bootstrap, libraryRoot: path }
      }
      await loadCard()
    } catch (err) {
      previewError.value = parseBackendError(err)
    }
  }

  async function chooseDump(dialogTitle: string) {
    try {
      const path = await window.go.handlers.App.SelectINPXFile(dialogTitle)
      if (!path) {
        return
      }
      await loadCard()
    } catch (err) {
      previewError.value = parseBackendError(err)
    }
  }

  async function pickDump(path: string) {
    try {
      await window.go.handlers.App.SetINPXPath(path)
      await loadCard()
    } catch (err) {
      previewError.value = parseBackendError(err)
    }
  }

  function requestStart() {
    if (!preview.value) {
      return
    }
    if (preview.value.hasCatalog) {
      confirmReimport.value = true
      return
    }
    void runImport()
  }

  async function runImport() {
    confirmReimport.value = false
    confirmCancel.value = false
    listen()
    modalOpen.value = true
    modalStage.value = 'progress'
    running.value = true
    result.value = null
    runError.value = null
    progress.value = {
      phase: 'records',
      recordsSeen: 0,
      bytesDone: 0,
      bytesTotal: 0,
      committed: false,
    }
    progressStartedAt.value = Date.now()
    try {
      const report = await window.go.handlers.App.StartImport()
      result.value = report
      lastReport.value = report
      modalStage.value = 'report'
      await useAppStore().refresh()
      useCatalogStore().reset()
      await loadCard()
    } catch (err) {
      const parsed = parseBackendError(err)
      await loadCard()
      if (parsed.code === 'import_cancelled') {
        result.value = lastReport.value
        runError.value = isImportReport(lastReport.value) ? null : parsed
        modalStage.value = 'report'
        return
      }
      runError.value = parsed
      result.value = null
      modalStage.value = 'report'
    } finally {
      running.value = false
    }
  }

  async function requestCancel() {
    if (!canCancel.value) {
      return
    }
    confirmCancel.value = true
  }

  async function confirmCancelImport() {
    confirmCancel.value = false
    await window.go.handlers.App.CancelImport()
  }

  function closeModal() {
    if (running.value) {
      return
    }
    modalOpen.value = false
    confirmCancel.value = false
  }

  function stayInApp() {
    confirmClose.value = false
    if (!closingApp.value) {
      void window.go.handlers.App.DismissWindowClose()
    }
  }

  async function leaveApp() {
    closingApp.value = true
    confirmClose.value = false
    if (!closeCommitted.value) {
      await window.go.handlers.App.CancelImport()
    }
    quitApp()
  }

  function runErrorText() {
    if (!runError.value) {
      return ''
    }
    return errorMessage(runError.value.code, runError.value.params)
  }

  return {
    cardLoading,
    preview,
    previewError,
    lastReport,
    modalOpen,
    modalStage,
    running,
    progress,
    progressStartedAt,
    result,
    runError,
    confirmReimport,
    confirmCancel,
    confirmClose,
    closeCommitted,
    libraryRoot,
    hasFolder,
    visiblePhase,
    canCancel,
    listen,
    loadCard,
    loadLastReport,
    chooseFolder,
    chooseDump,
    pickDump,
    requestStart,
    runImport,
    requestCancel,
    confirmCancelImport,
    closeModal,
    stayInApp,
    leaveApp,
    runErrorText,
  }
})
