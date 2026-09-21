<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import { authorParts, coverHue, isBlankTitle, isUnknownAuthor, titleMonogram } from '@/lib/work'
import { useAppStore } from '@/stores/app'
import { useCoversStore } from '@/stores/covers'
import { useStorageStore } from '@/stores/storage'
import type { Work } from '@/types/catalog'

const COVER_PRIORITY_HEADER = 'X-Cover-Priority'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    work: Pick<Work, 'id' | 'workKey' | 'title' | 'authorsText' | 'hasFile'>
    prio?: 'open' | 'visible' | 'prefetch'
    observe?: boolean
    compact?: boolean
    fit?: 'cover' | 'contain'
    class?: HTMLAttributes['class']
  }>(),
  { prio: 'visible', observe: true, compact: false, fit: 'cover' },
)

const { t } = useI18n()
const app = useAppStore()
const covers = useCoversStore()
const storage = useStorageStore()
const root = ref<HTMLElement | null>(null)
const status = ref<'plate' | 'loading' | 'image'>('plate')
const slow = ref(false)
const src = ref('')
let observer: IntersectionObserver | null = null
let intersecting = false
let inflight: AbortController | null = null
let inflightPrio = ''
let blobURL = ''
let absent = false
let temporary = false
let slowTimer: ReturnType<typeof setTimeout> | null = null

const title = computed(() =>
  isBlankTitle(props.work.title) ? t('catalog.untitled') : props.work.title,
)
const authors = computed(() => {
  const named = authorParts(props.work.authorsText).filter((name) => !isUnknownAuthor(name))
  return named.length ? named.join(', ') : t('catalog.unknownAuthor')
})
const hue = computed(() => coverHue(props.work.workKey || String(props.work.id)))
const monogram = computed(() => titleMonogram(title.value))
const canFetch = computed(() =>
  Boolean(app.bootstrap?.mediaBase && storage.available && props.work.hasFile && props.work.id),
)

function coverURL() {
  return `${app.bootstrap?.mediaBase}/media/cover/${props.work.id}`
}

function abortInflight() {
  inflight?.abort()
  inflight = null
  inflightPrio = ''
}

function revokeBlob() {
  if (blobURL) {
    URL.revokeObjectURL(blobURL)
    blobURL = ''
  }
}

function clearSlow() {
  if (slowTimer) {
    clearTimeout(slowTimer)
    slowTimer = null
  }
  slow.value = false
}

function markLoading() {
  status.value = 'loading'
  clearSlow()
  slowTimer = setTimeout(() => {
    slow.value = true
  }, 1000)
}

function showPlate() {
  revokeBlob()
  src.value = ''
  clearSlow()
  status.value = 'plate'
}

function prioRank(prio: string): number {
  if (prio === 'open') {
    return 0
  }
  if (prio === 'visible') {
    return 1
  }
  return 2
}

function load(prio: string) {
  if (!canFetch.value || absent) {
    showPlate()
    return
  }
  if (status.value === 'image' && src.value) {
    return
  }
  if (inflight && prioRank(prio) >= prioRank(inflightPrio)) {
    return
  }
  abortInflight()
  inflightPrio = prio
  const ac = new AbortController()
  inflight = ac
  if (status.value !== 'loading') {
    markLoading()
  }
  void (async () => {
    try {
      const resp = await fetch(coverURL(), {
        headers: { [COVER_PRIORITY_HEADER]: prio },
        cache: 'no-cache',
        signal: ac.signal,
      })
      if (ac.signal.aborted) {
        return
      }
      if (resp.status === 404) {
        absent = true
        temporary = false
        showPlate()
        return
      }
      if (!resp.ok) {
        temporary = true
        showPlate()
        return
      }
      const blob = await resp.blob()
      if (ac.signal.aborted) {
        return
      }
      revokeBlob()
      blobURL = URL.createObjectURL(blob)
      src.value = blobURL
      clearSlow()
      status.value = 'image'
      temporary = false
    } catch {
      if (ac.signal.aborted) {
        return
      }
      temporary = true
      showPlate()
    } finally {
      if (inflight === ac) {
        inflight = null
      }
    }
  })()
}

function onFailed() {
  showPlate()
}

function onIntersect(entries: IntersectionObserverEntry[]) {
  const entry = entries[0]
  if (!entry) {
    return
  }
  if (!entry.isIntersecting) {
    intersecting = false
    abortInflight()
    if (status.value !== 'image') {
      showPlate()
    }
    return
  }
  intersecting = true
  if (status.value === 'image' && src.value) {
    return
  }
  if (absent || temporary) {
    showPlate()
    return
  }
  const rect = entry.boundingClientRect
  const viewH = entry.rootBounds?.height ?? window.innerHeight
  const visible = rect.bottom > 0 && rect.top < viewH
  load(visible ? 'visible' : 'prefetch')
}

watch(
  () => [props.work.id, covers.generation] as const,
  () => {
    absent = false
    temporary = false
    abortInflight()
    showPlate()
    if (!canFetch.value) {
      return
    }
    if (!props.observe || intersecting) {
      load(props.observe ? 'visible' : props.prio)
    }
  },
)

watch(canFetch, (ok) => {
  if (!ok) {
    abortInflight()
    showPlate()
    return
  }
  if (absent || status.value === 'image') {
    return
  }
  if (!props.observe || intersecting) {
    load(props.observe ? 'visible' : props.prio)
  }
})

onMounted(() => {
  if (!props.observe) {
    load(props.prio)
    return
  }
  const node = root.value
  if (!node || typeof IntersectionObserver === 'undefined') {
    load(props.prio)
    return
  }
  observer = new IntersectionObserver(onIntersect, { rootMargin: '100% 0px', threshold: 0.01 })
  observer.observe(node)
})

onUnmounted(() => {
  observer?.disconnect()
  abortInflight()
  clearSlow()
  revokeBlob()
  src.value = ''
})
</script>

<template>
  <div ref="root" :class="cn('relative isolate overflow-hidden bg-muted', props.class)">
    <div
      class="flex h-full w-full items-center justify-center text-primary-foreground"
      :style="{ background: `hsl(${hue} 32% var(--cover-l, 28%))` }"
    >
      <slot v-if="status !== 'image'" name="plate" :title="title" :authors="authors">
        <span
          class="font-display font-semibold tracking-wide select-none"
          :class="compact ? 'text-sm' : 'text-5xl'"
          aria-hidden="true"
        >
          {{ monogram }}
        </span>
      </slot>
    </div>
    <Skeleton
      v-if="status === 'loading' && !slow"
      class="absolute inset-0 rounded-none"
      aria-hidden="true"
    />
    <p
      v-if="status === 'loading' && slow"
      class="absolute inset-0 z-10 flex items-center justify-center bg-background/70 px-3 text-center text-sm text-foreground"
    >
      {{ t('book.readingDisk') }}
    </p>
    <div
      v-if="src && fit === 'contain'"
      class="absolute inset-0 overflow-hidden bg-black"
      aria-hidden="true"
    >
      <img :src="src" alt="" class="h-full w-full scale-110 object-cover blur-2xl brightness-50" />
    </div>
    <img
      v-if="src"
      :src="src"
      alt=""
      class="absolute inset-0 h-full w-full"
      :class="fit === 'contain' ? 'object-contain' : 'object-cover'"
      :style="status === 'image' ? undefined : { opacity: 0 }"
      @error="onFailed"
    />
  </div>
</template>
