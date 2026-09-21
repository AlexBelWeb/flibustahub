import { defineStore } from 'pinia'
import { ref } from 'vue'
import { errorMessage } from '@/i18n/errors'
import { parseBackendError, type BackendError } from '@/lib/backend-error'
import { useAppStore } from '@/stores/app'
import type { AIProviderId, SecretStatus } from '@/types/secrets'

export const useSecretsStore = defineStore('secrets', () => {
  const status = ref<SecretStatus | null>(null)
  const loadState = ref<'idle' | 'loading' | 'ready' | 'empty' | 'error'>('idle')
  const loadError = ref<BackendError | null>(null)
  const saving = ref(false)
  const saveError = ref<BackendError | null>(null)

  async function load(id: string) {
    loadState.value = 'loading'
    loadError.value = null
    try {
      status.value = await window.go.handlers.App.SecretStatus(id)
      loadState.value = status.value ? 'ready' : 'empty'
    } catch (err) {
      status.value = null
      loadError.value = parseBackendError(err)
      loadState.value = 'error'
    }
  }

  async function setProvider(id: string) {
    await window.go.handlers.App.SetAIProvider(id)
    const app = useAppStore()
    if (app.bootstrap) {
      app.bootstrap = { ...app.bootstrap, aiProvider: id }
    }
  }

  async function saveSecret(id: AIProviderId, secret: string) {
    saveError.value = null
    saving.value = true
    try {
      await window.go.handlers.App.SetSecret(id, secret)
      await load(id)
    } catch (err) {
      saveError.value = parseBackendError(err)
      throw err
    } finally {
      saving.value = false
    }
  }

  async function deleteSecret(id: string) {
    saveError.value = null
    saving.value = true
    try {
      await window.go.handlers.App.DeleteSecret(id)
      await load(id)
    } catch (err) {
      saveError.value = parseBackendError(err)
      throw err
    } finally {
      saving.value = false
    }
  }

  function errorText(err: BackendError | null) {
    if (!err) {
      return ''
    }
    return errorMessage(err.code, err.params)
  }

  return {
    status,
    loadState,
    loadError,
    saving,
    saveError,
    load,
    setProvider,
    saveSecret,
    deleteSecret,
    errorText,
  }
})
