import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { errorMessage } from '@/i18n/errors'
import { Events } from '@/lib/events'
import { parseBackendError, type BackendError } from '@/lib/backend-error'
import { eventsOn, quitApp } from '@/lib/wails-runtime'
import type { BackupResult, OptimizeResult } from '@/types/maintenance'

export const useMaintenanceStore = defineStore('maintenance', () => {
  const running = ref(false)
  const confirmClose = ref(false)
  const closingApp = ref(false)
  const lastError = ref<BackendError | null>(null)
  const optimizeResult = ref<OptimizeResult | null>(null)
  const backupResult = ref<BackupResult | null>(null)

  const busy = computed(() => running.value)

  let listening = false

  function listen() {
    if (listening) {
      return
    }
    listening = true
    eventsOn(Events.MaintenanceCloseRequested, () => {
      confirmClose.value = true
    })
  }

  async function refreshRunning() {
    running.value = await window.go.handlers.App.DatabaseMaintenanceRunning()
  }

  async function optimize(): Promise<OptimizeResult | null> {
    lastError.value = null
    optimizeResult.value = null
    running.value = true
    try {
      const result = await window.go.handlers.App.OptimizeDatabase()
      optimizeResult.value = result
      backupResult.value = null
      return result
    } catch (err) {
      lastError.value = parseBackendError(err)
      throw err
    } finally {
      running.value = false
    }
  }

  async function backup(): Promise<BackupResult | null> {
    lastError.value = null
    backupResult.value = null
    running.value = true
    try {
      const result = await window.go.handlers.App.CreateCatalogBackup()
      if (result?.path) {
        backupResult.value = result
      }
      return result
    } catch (err) {
      lastError.value = parseBackendError(err)
      throw err
    } finally {
      running.value = false
    }
  }

  function stayInApp() {
    confirmClose.value = false
    if (!closingApp.value) {
      void window.go.handlers.App.DismissWindowClose()
    }
  }

  function leaveApp() {
    closingApp.value = true
    confirmClose.value = false
    quitApp()
  }

  function errorText() {
    if (!lastError.value) {
      return ''
    }
    return errorMessage(lastError.value.code, lastError.value.params)
  }

  return {
    running,
    busy,
    confirmClose,
    lastError,
    optimizeResult,
    backupResult,
    listen,
    refreshRunning,
    optimize,
    backup,
    stayInApp,
    leaveApp,
    errorText,
  }
})
