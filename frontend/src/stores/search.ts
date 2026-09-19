import { defineStore } from 'pinia'
import { ref } from 'vue'
import { parseBackendError, type BackendError } from '@/lib/backend-error'
import { asSearchResult, type SearchResult } from '@/types/catalog'

export const useSearchStore = defineStore('search', () => {
  const history = ref<string[]>([])
  const historyError = ref<BackendError | null>(null)
  const preview = ref<SearchResult | null>(null)
  const previewStatus = ref<'idle' | 'loading' | 'empty' | 'ready' | 'error'>('idle')
  const previewError = ref<BackendError | null>(null)
  let seq = 0

  async function loadHistory() {
    try {
      history.value = (await window.go.handlers.App.SearchHistory()) ?? []
      historyError.value = null
    } catch (err) {
      historyError.value = parseBackendError(err)
    }
  }

  async function record(query: string) {
    const q = query.trim()
    if (!q) {
      return
    }
    await window.go.handlers.App.RecordSearch(q)
    await loadHistory()
  }

  async function clearHistory() {
    await window.go.handlers.App.ClearSearchHistory()
    history.value = []
  }

  async function previewSearch(query: string, indexReady: boolean) {
    const q = query.trim()
    if (!q) {
      seq += 1
      preview.value = null
      previewStatus.value = 'idle'
      previewError.value = null
      return
    }
    if (!indexReady) {
      return
    }
    const token = ++seq
    previewStatus.value = 'loading'
    try {
      const result = asSearchResult(await window.go.handlers.App.SearchCatalog({ q, limit: 7 }))
      if (token !== seq) {
        return
      }
      preview.value = result
      const empty =
        result.works.items.length === 0 &&
        (result.authors?.length ?? 0) === 0 &&
        (result.series?.length ?? 0) === 0
      previewStatus.value = empty ? 'empty' : 'ready'
    } catch (err) {
      if (token !== seq) {
        return
      }
      previewError.value = parseBackendError(err)
      previewStatus.value = 'error'
    }
  }

  return {
    history,
    historyError,
    preview,
    previewStatus,
    previewError,
    loadHistory,
    record,
    clearHistory,
    previewSearch,
  }
})
