<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useVirtualizer } from '@tanstack/vue-virtual'
import { useI18n } from 'vue-i18n'
import WorkCard from '@/components/catalog/WorkCard.vue'
import WorkRow from '@/components/catalog/WorkRow.vue'
import { gridKeyHandled, nextGridIndex } from '@/lib/grid-nav'
import type { CatalogView, Work } from '@/types/catalog'

const props = defineProps<{
  items: Work[]
  view: CatalogView
  query?: string
  scrollTop: number
  loadingMore: boolean
  exhausted: boolean
  capped?: boolean
  totalN?: number
}>()

const emit = defineEmits<{
  end: []
  scroll: [top: number]
}>()

const { t } = useI18n()
const parentRef = ref<HTMLElement | null>(null)
const width = ref(900)
const focusedIndex = ref(-1)

const lanes = computed(() => {
  if (props.view !== 'tile') {
    return 1
  }
  if (width.value < 520) {
    return 2
  }
  if (width.value < 780) {
    return 3
  }
  if (width.value < 1100) {
    return 4
  }
  return 5
})

const estimate = computed(() => (props.view === 'tile' ? 500 : 48))

const virtualizer = useVirtualizer(
  computed(() => ({
    count: props.items.length,
    getScrollElement: () => parentRef.value,
    estimateSize: () => estimate.value,
    overscan: 8,
    lanes: lanes.value,
    initialOffset: props.scrollTop,
  })),
)

const lanePct = computed(() => 100 / lanes.value)

function measure() {
  virtualizer.value.measure()
}

function onScroll() {
  const el = parentRef.value
  if (!el) {
    return
  }
  emit('scroll', el.scrollTop)
  const last = virtualizer.value.getVirtualItems().at(-1)
  if (last && last.index >= props.items.length - lanes.value * 3) {
    emit('end')
  }
}

function focusIndex(index: number) {
  focusedIndex.value = index
  virtualizer.value.scrollToIndex(index, { align: 'auto' })
  void nextTick(() => {
    requestAnimationFrame(() => {
      const node = parentRef.value?.querySelector<HTMLElement>(`[data-work-index="${index}"]`)
      node?.focus()
    })
  })
}

function onKeydown(event: KeyboardEvent) {
  if (!gridKeyHandled(event.key) || !props.items.length) {
    return
  }
  if (event.key === 'Enter') {
    if (focusedIndex.value < 0) {
      return
    }
    const node = parentRef.value?.querySelector<HTMLElement>(
      `[data-work-index="${focusedIndex.value}"]`,
    )
    node?.click()
    event.preventDefault()
    return
  }
  const next = nextGridIndex(focusedIndex.value, event.key, props.items.length, lanes.value)
  if (next == null || next === focusedIndex.value) {
    return
  }
  event.preventDefault()
  focusIndex(next)
}

let ro: ResizeObserver | null = null

onMounted(() => {
  const el = parentRef.value
  if (!el) {
    return
  }
  width.value = el.clientWidth
  ro = new ResizeObserver(() => {
    width.value = el.clientWidth
    measure()
  })
  ro.observe(el)
  if (props.scrollTop) {
    el.scrollTop = props.scrollTop
  }
  void nextTick(() => measure())
})

onUnmounted(() => {
  ro?.disconnect()
  if (parentRef.value) {
    emit('scroll', parentRef.value.scrollTop)
  }
})

watch(
  () => [props.items.length, props.view, lanes.value] as const,
  () => {
    void nextTick(() => measure())
    const last = virtualizer.value.getVirtualItems().at(-1)
    if (last && last.index >= props.items.length - lanes.value * 3) {
      emit('end')
    }
  },
)
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col">
    <div
      v-if="view === 'table'"
      class="grid grid-cols-[2.5rem_minmax(0,2fr)_minmax(0,1.5fr)_minmax(0,1fr)_4rem_6rem_4rem] gap-3 border-b border-border px-2 py-2 text-xs text-muted-foreground"
    >
      <span />
      <span>{{ t('catalog.colTitle') }}</span>
      <span>{{ t('catalog.colAuthors') }}</span>
      <span>{{ t('catalog.colSeries') }}</span>
      <span>{{ t('catalog.colLang') }}</span>
      <span class="text-right">{{ t('catalog.colSize') }}</span>
      <span class="text-right">{{ t('catalog.colRating') }}</span>
    </div>
    <div class="relative min-h-0 flex-1">
      <div
        ref="parentRef"
        class="absolute inset-0 overflow-auto"
        tabindex="0"
        @scroll.passive="onScroll"
        @keydown="onKeydown"
      >
        <div class="relative w-full" :style="{ height: `${virtualizer.getTotalSize()}px` }">
          <div
            v-for="row in virtualizer.getVirtualItems()"
            :key="row.key"
            class="absolute top-0"
            :style="{
              transform: `translateY(${row.start}px)`,
              height: `${row.size}px`,
              left: view === 'tile' ? `${row.lane * lanePct}%` : '0',
              width: view === 'tile' ? `${lanePct}%` : '100%',
              padding: view === 'tile' ? '0.5rem' : '0',
            }"
          >
            <WorkCard
              v-if="view === 'tile' && items[row.index]"
              :work="items[row.index]"
              :query="query"
              :data-work-index="row.index"
              :tabindex="focusedIndex === row.index ? 0 : -1"
              @focus="focusedIndex = row.index"
            />
            <WorkRow
              v-else-if="items[row.index]"
              :work="items[row.index]"
              :query="query"
              :data-work-index="row.index"
              :tabindex="focusedIndex === row.index ? 0 : -1"
              @focus="focusedIndex = row.index"
            />
          </div>
        </div>
        <p v-if="loadingMore" class="p-4 text-center text-sm text-muted-foreground">
          {{ t('list.more') }}
        </p>
        <p v-else-if="capped" class="p-4 text-center text-sm text-muted-foreground">
          {{ t('list.searchLimit', { n: totalN ?? 500 }) }}
        </p>
        <p
          v-else-if="exhausted && items.length > 0"
          class="p-4 text-center text-sm text-muted-foreground"
        >
          {{ t('list.end') }}
        </p>
      </div>
    </div>
  </div>
</template>
