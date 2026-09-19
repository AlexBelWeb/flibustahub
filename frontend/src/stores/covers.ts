import { defineStore } from 'pinia'
import { ref } from 'vue'
import { Events } from '@/lib/events'
import { eventsOn } from '@/lib/wails-runtime'
import type { CoverProgress } from '@/types/catalog'

export const useCoversStore = defineStore('covers', () => {
  const generation = ref(0)
  const progress = ref<CoverProgress>({ total: 0, done: 0, running: false })
  let listening = false

  function eventRecord(data: unknown): Record<string, unknown> {
    if (Array.isArray(data) && data.length > 0) {
      return eventRecord(data[0])
    }
    if (data && typeof data === 'object') {
      return data as Record<string, unknown>
    }
    return {}
  }

  function listen() {
    if (listening) {
      return
    }
    listening = true
    eventsOn(Events.CoversProgress, (data) => {
      const src = eventRecord(data)
      progress.value = {
        total: typeof src.total === 'number' ? src.total : 0,
        done: typeof src.done === 'number' ? src.done : 0,
        running: Boolean(src.running),
      }
    })
  }

  async function preview(): Promise<number> {
    return window.go.handlers.App.CoverWarmupPreview()
  }

  async function start() {
    listen()
    await window.go.handlers.App.StartCoverWarmup()
    progress.value = await window.go.handlers.App.CoverWarmupProgress()
  }

  function stop() {
    window.go.handlers.App.StopCoverWarmup()
  }

  async function clear() {
    await window.go.handlers.App.ClearCoverCache()
    generation.value += 1
  }

  return { generation, progress, listen, preview, start, stop, clear }
})
