<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Skeleton } from '@/components/ui/skeleton'
import { authorParts, coverHue, isBlankTitle } from '@/lib/work'
import { useAppStore } from '@/stores/app'
import { useCoversStore } from '@/stores/covers'
import type { Work } from '@/types/catalog'

const COVER_PRIORITY_HEADER = 'X-Cover-Priority'

const props = withDefaults(
  defineProps<{
    work: Pick<Work, 'id' | 'workKey' | 'title' | 'authorsText' | 'hasFile'>
    prio?: 'open' | 'visible' | 'prefetch'
    observe?: boolean
    compact?: boolean
  }>(),
  { prio: 'visible', observe: true, compact: false },
)

const { t } = useI18n()
const app = useAppStore()
const covers = useCoversStore()
const root = ref<HTMLElement | null>(null)
const status = ref<'plate' | 'loading' | 'image'>('plate')
const src = ref('')
let observer: IntersectionObserver | null = null
let intersecting = false
let inflight: AbortController | null = null
let blobURL = ''
let absent = false
let temporary = false

const title = computed(() =>
  isBlankTitle(props.work.title) ? t('catalog.untitled') : props.work.title,
)
const authors = computed(() => authorParts(props.work.authorsText).join(', '))
const hue = computed(() => coverHue(props.work.workKey || String(props.work.id)))
const canFetch = computed(() =>
  Boolean(
    app.bootstrap?.mediaBase && app.bootstrap.libraryRoot && props.work.hasFile && props.work.id,
  ),
)

function coverURL() {
  return `${app.bootstrap?.mediaBase}/media/cover/${props.work.id}`
}

function abortInflight() {
  inflight?.abort()
  inflight = null
}

function revokeBlob() {
  if (blobURL) {
    URL.revokeObjectURL(blobURL)
    blobURL = ''
  }
}

function showPlate() {
  revokeBlob()
  src.value = ''
  status.value = 'plate'
}

function load(prio: string) {
  if (!canFetch.value || absent) {
    showPlate()
    return
  }
  abortInflight()
  const ac = new AbortController()
  inflight = ac
  status.value = 'loading'
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
  revokeBlob()
  src.value = ''
})
</script>

<template>
  <div ref="root" class="relative overflow-hidden bg-muted">
    <div
      class="flex h-full w-full flex-col justify-end text-primary-foreground"
      :class="compact ? 'p-0' : 'p-3'"
      :style="{ background: `hsl(${hue} 32% var(--cover-l, 28%))` }"
    >
      <slot v-if="!compact" name="plate" :title="title" :authors="authors">
        <p class="line-clamp-2 font-display text-sm font-semibold">{{ title }}</p>
        <p v-if="authors" class="mt-1 line-clamp-1 text-xs opacity-90">{{ authors }}</p>
      </slot>
    </div>
    <Skeleton
      v-if="status === 'loading'"
      class="absolute inset-0 rounded-none"
      aria-hidden="true"
    />
    <img
      v-if="src"
      :src="src"
      alt=""
      class="absolute inset-0 h-full w-full object-cover"
      :class="status === 'image' ? 'opacity-100' : 'opacity-0'"
      @error="onFailed"
    />
  </div>
</template>
