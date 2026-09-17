import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

export type ToastKind = 'info' | 'error'

export interface ToastItem {
  id: number
  kind: ToastKind
  message: string
  count: number
}

let nextId = 1
const INFO_TTL_MS = 4000
const MAX_STACK = 3

export const useToastStore = defineStore('toast', () => {
  const items = ref<ToastItem[]>([])
  const timers = new Map<number, number>()

  const visible = computed(() => items.value.slice(-MAX_STACK))

  function dismiss(id: number) {
    const timer = timers.get(id)
    if (timer) {
      window.clearTimeout(timer)
      timers.delete(id)
    }
    items.value = items.value.filter((item) => item.id !== id)
  }

  function push(kind: ToastKind, message: string) {
    const existing = items.value.find((item) => item.kind === kind && item.message === message)
    if (existing) {
      existing.count += 1
      if (kind !== 'error') {
        restartTimer(existing.id)
      }
      return
    }
    const id = nextId++
    items.value.push({ id, kind, message, count: 1 })
    if (items.value.length > MAX_STACK) {
      const dropped = items.value.shift()
      if (dropped) {
        dismiss(dropped.id)
      }
    }
    if (kind !== 'error') {
      restartTimer(id)
    }
  }

  function restartTimer(id: number) {
    const prev = timers.get(id)
    if (prev) {
      window.clearTimeout(prev)
    }
    timers.set(
      id,
      window.setTimeout(() => dismiss(id), INFO_TTL_MS),
    )
  }

  function pushError(message: string) {
    push('error', message)
  }

  function pushInfo(message: string) {
    push('info', message)
  }

  return { items, visible, dismiss, pushError, pushInfo }
})
