import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

export type ToastKind = 'info' | 'error'

export interface ToastItem {
  id: number
  kind: ToastKind
  message: string
  count: number
  actionLabel?: string
  onAction?: () => void
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

  function push(kind: ToastKind, message: string, action?: { label: string; run: () => void }) {
    const existing = items.value.find((item) => item.kind === kind && item.message === message)
    if (existing) {
      existing.count += 1
      existing.actionLabel = action?.label
      existing.onAction = action?.run
      if (kind !== 'error') {
        restartTimer(existing.id)
      }
      return
    }
    const id = nextId++
    items.value.push({
      id,
      kind,
      message,
      count: 1,
      actionLabel: action?.label,
      onAction: action?.run,
    })
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

  function pushError(message: string, action?: { label: string; run: () => void }) {
    push('error', message, action)
  }

  function pushInfo(message: string) {
    push('info', message)
  }

  function runAction(id: number) {
    const item = items.value.find((entry) => entry.id === id)
    const run = item?.onAction
    dismiss(id)
    run?.()
  }

  return { items, visible, dismiss, pushError, pushInfo, runAction }
})
