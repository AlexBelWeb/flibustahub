<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ListState from '@/components/catalog/ListState.vue'
import WorkCard from '@/components/catalog/WorkCard.vue'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { errorMessage } from '@/i18n/errors'
import { formatCount, formatDate, formatRelative } from '@/lib/format'
import { gridKeyHandled, nextGridIndex } from '@/lib/grid-nav'
import { HOME_ARRIVALS_KEY, useCatalogStore } from '@/stores/catalog'
import { useImportStore } from '@/stores/import'
import { useUiStore } from '@/stores/ui'

const { t, locale } = useI18n()
const catalog = useCatalogStore()
const imp = useImportStore()
const ui = useUiStore()
const gridRef = ref<HTMLElement | null>(null)
const focusedIndex = ref(-1)
const gridWidth = ref(900)

const state = computed(() => catalog.workState(HOME_ARRIVALS_KEY))
const books = computed(() => {
  const n = state.value.totalN
  if (n == null || state.value.capped) {
    return ''
  }
  return formatCount(n, locale.value)
})

const importLine = computed(() => {
  const report = imp.lastReport
  if (!report?.finishedAt && !report?.inpxVersion) {
    return t('home.lastImportNever')
  }
  const version = report.inpxVersion
    ? t('home.lastImportVersion', { version: report.inpxVersion })
    : ''
  const when = report.finishedAt ? formatRelative(report.finishedAt, locale.value) : ''
  return [when, version].filter(Boolean).join(' · ')
})

const importExact = computed(() =>
  imp.lastReport?.finishedAt ? formatDate(imp.lastReport.finishedAt, locale.value) : '',
)

const emptyCatalog = computed(
  () =>
    state.value.status === 'empty' ||
    (state.value.status === 'ready' && (state.value.totalN ?? 0) === 0),
)

const columns = computed(() => {
  if (gridWidth.value >= 1024) {
    return 6
  }
  if (gridWidth.value >= 768) {
    return 4
  }
  if (gridWidth.value >= 640) {
    return 3
  }
  return 2
})

function onGridKey(event: KeyboardEvent) {
  if (!gridKeyHandled(event.key) || !state.value.items.length) {
    return
  }
  if (event.key === 'Enter') {
    if (focusedIndex.value < 0) {
      return
    }
    gridRef.value?.querySelector<HTMLElement>(`[data-work-index="${focusedIndex.value}"]`)?.click()
    event.preventDefault()
    return
  }
  const next = nextGridIndex(focusedIndex.value, event.key, state.value.items.length, columns.value)
  if (next == null) {
    return
  }
  event.preventDefault()
  focusedIndex.value = next
  gridRef.value?.querySelector<HTMLElement>(`[data-work-index="${next}"]`)?.focus()
}

async function load() {
  await catalog.loadWorks(HOME_ARRIVALS_KEY, { sort: 'added', limit: 18 }, true)
  await imp.loadLastReport()
}

watch(emptyCatalog, (empty) => {
  if (empty && !ui.onboardingDismissed) {
    ui.openOnboarding()
  }
})

let ro: ResizeObserver | null = null

function attachGridObserver(el: HTMLElement | null) {
  ro?.disconnect()
  ro = null
  if (!el) {
    return
  }
  gridWidth.value = el.clientWidth
  ro = new ResizeObserver(() => {
    gridWidth.value = el.clientWidth
  })
  ro.observe(el)
}

onMounted(() => {
  if (state.value.status === 'idle' || state.value.status === 'error') {
    void load()
  } else if (emptyCatalog.value && !ui.onboardingDismissed) {
    ui.openOnboarding()
  }
})

watch(gridRef, (el) => attachGridObserver(el))

onUnmounted(() => {
  ro?.disconnect()
  ro = null
})

watch(
  () => catalog.works[HOME_ARRIVALS_KEY]?.status ?? 'idle',
  (status, prev) => {
    if (status === 'idle' && prev && prev !== 'idle') {
      void load()
    }
  },
)
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col overflow-auto px-6 pt-8">
    <ListState
      class="flex-none"
      :status="state.status === 'ready' && emptyCatalog ? 'empty' : state.status"
      :empty-text="t('home.emptyTitle')"
      :error-text="
        state.error ? errorMessage(state.error.code, state.error.params) : t('list.error')
      "
      @retry="load"
    >
      <template #empty>
        <p class="mt-2 text-muted-foreground">{{ t('home.emptyBody') }}</p>
        <Button class="mt-4 w-fit" @click="ui.openOnboarding()">{{
          t('home.continueSetup')
        }}</Button>
      </template>

      <div class="grid gap-8 pb-12">
        <section class="grid gap-4 sm:grid-cols-2">
          <Card class="p-6">
            <p class="text-sm text-muted-foreground">{{ t('home.books') }}</p>
            <p class="mt-2 font-display text-4xl tabular-nums">{{ books }}</p>
          </Card>
          <Card class="p-6">
            <p class="text-sm text-muted-foreground">{{ t('home.lastImport') }}</p>
            <Tooltip :disabled="!importExact">
              <TooltipTrigger as-child>
                <p class="mt-2 text-lg">{{ importLine }}</p>
              </TooltipTrigger>
              <TooltipContent v-if="importExact">{{ importExact }}</TooltipContent>
            </Tooltip>
          </Card>
        </section>

        <section class="grid gap-4">
          <h2 class="font-display text-2xl font-semibold">{{ t('home.newArrivals') }}</h2>
          <div
            ref="gridRef"
            class="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6"
            tabindex="0"
            @keydown="onGridKey"
          >
            <WorkCard
              v-for="(work, index) in state.items"
              :key="work.id"
              :work="work"
              :data-work-index="index"
              :tabindex="focusedIndex === index ? 0 : -1"
              @focus="focusedIndex = index"
            />
          </div>
        </section>
      </div>
    </ListState>
  </div>
</template>
