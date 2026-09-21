<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useVirtualizer } from '@tanstack/vue-virtual'
import { useI18n } from 'vue-i18n'
import WorkCard from '@/components/catalog/WorkCard.vue'
import WorkRow from '@/components/catalog/WorkRow.vue'
import { SCROLL_END_PAD } from '@/lib/scroll'
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

const rowCount = computed(() =>
  props.view === 'tile' ? Math.ceil(props.items.length / lanes.value) : props.items.length,
)

// Guess only. The virtualizer replaces it with the rendered row height.
function estimateTileRow(): number {
  const col = width.value / Math.max(1, lanes.value)
  return Math.round(Math.max(col, 160) * 2)
}

const virtualizer = useVirtualizer(
  computed(() => ({
    count: rowCount.value,
    getScrollElement: () => parentRef.value,
    estimateSize: () => (props.view === 'tile' ? estimateTileRow() : 48),
    // One row each side keeps the next keyboard step mounted without a screen of extra cards.
    overscan: props.view === 'tile' ? 1 : 8,
    initialOffset: props.scrollTop,
  })),
)

function measureRow(node: Element | null) {
  virtualizer.value.measureElement(node)
}

function indexesInRow(rowIndex: number): number[] {
  const start = rowIndex * lanes.value
  const end = Math.min(start + lanes.value, props.items.length)
  const indexes: number[] = []
  for (let index = start; index < end; index += 1) {
    indexes.push(index)
  }
  return indexes
}

function nearEnd(): boolean {
  const last = virtualizer.value.getVirtualItems().at(-1)
  if (!last) {
    return false
  }
  return last.index >= rowCount.value - 3
}

function onScroll() {
  const el = parentRef.value
  if (!el) {
    return
  }
  emit('scroll', el.scrollTop)
  if (nearEnd()) {
    emit('end')
  }
}

function focusIndex(index: number) {
  focusedIndex.value = index
  const row = props.view === 'tile' ? Math.floor(index / lanes.value) : index
  virtualizer.value.scrollToIndex(row, { align: 'auto' })
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
  })
  ro.observe(el)
  if (props.scrollTop) {
    el.scrollTop = props.scrollTop
  }
})

onUnmounted(() => {
  ro?.disconnect()
  if (parentRef.value) {
    emit('scroll', parentRef.value.scrollTop)
  }
})

watch(lanes, () => {
  void nextTick(() => {
    virtualizer.value.measure()
    parentRef.value?.querySelectorAll<HTMLElement>('[data-index]').forEach((node) => {
      virtualizer.value.measureElement(node)
    })
  })
})

watch(
  () => props.items.length,
  () => {
    if (nearEnd()) {
      emit('end')
    }
  },
)
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col">
    <div
      v-if="view === 'table'"
      class="grid grid-cols-[2.5rem_minmax(0,2fr)_minmax(0,1.5fr)_minmax(0,1fr)_7rem_6rem_4rem] gap-3 border-b border-border px-2 py-2 text-xs text-muted-foreground"
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
        data-catalog-scroll
        tabindex="0"
        @scroll.passive="onScroll"
        @keydown="onKeydown"
      >
        <div
          class="relative w-full"
          :style="{ height: `${virtualizer.getTotalSize() + SCROLL_END_PAD}px` }"
        >
          <div
            v-for="row in virtualizer.getVirtualItems()"
            :key="row.key"
            :ref="measureRow"
            :data-index="row.index"
            class="absolute top-0 left-0 w-full"
            :style="{ transform: `translateY(${row.start}px)` }"
          >
            <div
              v-if="view === 'tile'"
              class="grid items-stretch"
              :style="{ gridTemplateColumns: `repeat(${lanes}, minmax(0, 1fr))` }"
            >
              <div v-for="index in indexesInRow(row.index)" :key="index" class="flex h-full p-2">
                <WorkCard
                  v-if="items[index]"
                  :work="items[index]"
                  :query="query"
                  class="h-full min-w-0 w-full flex-1"
                  :data-work-index="index"
                  :tabindex="focusedIndex === index ? 0 : -1"
                  @focus="focusedIndex = index"
                />
              </div>
            </div>
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
