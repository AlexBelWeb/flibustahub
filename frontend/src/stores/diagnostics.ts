import { defineStore } from 'pinia'
import { ref } from 'vue'
import { errorMessage } from '@/i18n/errors'
import { parseBackendError, type BackendError } from '@/lib/backend-error'
import type { ArchiveResult, DiagnosticSnapshot, IssueKind, IssueReport } from '@/types/diagnostics'

export const useDiagnosticsStore = defineStore('diagnostics', () => {
  const snapshot = ref<DiagnosticSnapshot | null>(null)
  const status = ref<'idle' | 'loading' | 'ready' | 'empty' | 'error'>('idle')
  const error = ref<BackendError | null>(null)
  const archiveBusy = ref(false)
  const archivePath = ref('')
  const archiveError = ref<BackendError | null>(null)

  async function load() {
    status.value = 'loading'
    error.value = null
    try {
      snapshot.value = await window.go.handlers.App.Diagnostics()
      status.value = snapshot.value ? 'ready' : 'empty'
    } catch (err) {
      snapshot.value = null
      error.value = parseBackendError(err)
      status.value = 'error'
    }
  }

  async function saveArchive(title: string): Promise<ArchiveResult | null> {
    archiveError.value = null
    archiveBusy.value = true
    try {
      const result = await window.go.handlers.App.SaveDiagnosticArchive(title)
      if (result?.path) {
        archivePath.value = result.path
      }
      return result
    } catch (err) {
      archiveError.value = parseBackendError(err)
      throw err
    } finally {
      archiveBusy.value = false
    }
  }

  async function buildIssue(kind: IssueKind, description: string): Promise<IssueReport> {
    return window.go.handlers.App.BuildIssueReport(kind, description)
  }

  async function saveIssue(title: string, body: string): Promise<ArchiveResult> {
    return window.go.handlers.App.SaveIssueReport(title, body)
  }

  function errorText() {
    if (!error.value) {
      return ''
    }
    return errorMessage(error.value.code, error.value.params)
  }

  function archiveErrorText() {
    if (!archiveError.value) {
      return ''
    }
    return errorMessage(archiveError.value.code, archiveError.value.params)
  }

  return {
    snapshot,
    status,
    error,
    archiveBusy,
    archivePath,
    archiveError,
    load,
    saveArchive,
    buildIssue,
    saveIssue,
    errorText,
    archiveErrorText,
  }
})
