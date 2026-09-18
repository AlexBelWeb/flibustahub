<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Funnel, X } from '@lucide/vue'
import EntityFilter from '@/components/catalog/EntityFilter.vue'
import ListState from '@/components/catalog/ListState.vue'
import VirtualWorkList from '@/components/catalog/VirtualWorkList.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { errorMessage } from '@/i18n/errors'
import { formatCount } from '@/lib/format'
import { queryId, queryText } from '@/lib/route-query'
import { useAppStore } from '@/stores/app'
import { useCatalogStore } from '@/stores/catalog'
import { useSearchStore } from '@/stores/search'
import { useUiStore } from '@/stores/ui'
import type { CatalogView } from '@/types/catalog'

const props = defineProps<{
  cacheKey: string
  authorId?: number
  genreId?: number
  seriesId?: number
  seriesOrder?: boolean
  showRandom?: boolean
}>()

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const app = useAppStore()
const catalog = useCatalogStore()
const search = useSearchStore()
const ui = useUiStore()

const searchInput = ref<{ focus: () => void } | null>(null)
const draft = ref('')
const filtersOpen = ref(false)

const q = computed(() => queryText(route.query.q).trim())
const lang = computed(() => queryText(route.query.lang))
const sort = computed(() => queryText(route.query.sort))
const genreFilter = computed(() => props.genreId || queryId(route.query.genre))
const authorFilter = computed(() => props.authorId || queryId(route.query.author))
const seriesFilter = computed(() => props.seriesId || queryId(route.query.series))

const listKey = computed(() =>
  JSON.stringify({
    k: props.cacheKey,
    q: q.value,
    lang: lang.value,
    sort: sort.value,
    genre: genreFilter.value,
    author: authorFilter.value,
    series: seriesFilter.value,
  }),
)

const state = computed(() => catalog.workState(listKey.value))
const searching = computed(() => q.value.length > 0)

function patchQuery(next: Record<string, string | undefined>) {
  const query: Record<string, string> = {}
  const merged = {
    q: q.value,
    lang: lang.value,
    sort: sort.value,
    genre: route.query.genre ? String(route.query.genre) : '',
    author: route.query.author ? String(route.query.author) : '',
    series: route.query.series ? String(route.query.series) : '',
    ...next,
  }
  for (const [key, value] of Object.entries(merged)) {
    if (value) {
      query[key] = value
    }
  }
  void router.replace({ query })
}

async function reload() {
  const key = listKey.value
  if (searching.value) {
    await catalog.loadSearch(
      key,
      {
        q: q.value,
        lang: lang.value || undefined,
        genreId: genreFilter.value || undefined,
        authorId: authorFilter.value || undefined,
        seriesId: seriesFilter.value || undefined,
      },
      true,
    )
    return
  }
  const nextSort = props.seriesOrder && !sort.value ? 'seriesno' : sort.value || 'title'
  await catalog.loadWorks(
    key,
    {
      sort: nextSort,
      lang: lang.value || undefined,
      genreId: genreFilter.value || undefined,
      authorId: authorFilter.value || undefined,
      seriesId: seriesFilter.value || undefined,
    },
    true,
  )
}

function loadMore() {
  const key = listKey.value
  const current = catalog.workState(key)
  if (current.status !== 'ready' || current.exhausted || current.loadingMore) {
    return
  }
  if (searching.value) {
    void catalog.loadSearch(
      key,
      {
        q: q.value,
        lang: lang.value || undefined,
        genreId: genreFilter.value || undefined,
        authorId: authorFilter.value || undefined,
        seriesId: seriesFilter.value || undefined,
      },
      false,
    )
    return
  }
  const nextSort = props.seriesOrder && !sort.value ? 'seriesno' : sort.value || 'title'
  void catalog.loadWorks(
    key,
    {
      sort: nextSort,
      lang: lang.value || undefined,
      genreId: genreFilter.value || undefined,
      authorId: authorFilter.value || undefined,
      seriesId: seriesFilter.value || undefined,
    },
    false,
  )
}

watch(
  listKey,
  () => {
    const current = catalog.workState(listKey.value)
    if (current.status === 'idle' || current.status === 'error') {
      void reload()
    }
  },
  { immediate: true },
)

watch(
  () => q.value,
  (value) => {
    draft.value = value
  },
  { immediate: true },
)

watch(
  () => ui.searchFocusNonce,
  () => {
    searchInput.value?.focus()
  },
)

let debounceTimer = 0

function onDraftInput(value: string) {
  draft.value = value
  if (!app.searchIndexReady) {
    return
  }
  window.clearTimeout(debounceTimer)
  debounceTimer = window.setTimeout(() => {
    commitSearch(value, false)
  }, 250)
}

function commitSearch(value: string, record: boolean) {
  const next = value.trim()
  if (next === q.value) {
    return
  }
  if (record && next) {
    void search.record(next)
  }
  patchQuery({
    q: next || undefined,
    sort: next && !sort.value ? 'relevance' : sort.value || undefined,
  })
}

function onSearchKey(event: KeyboardEvent) {
  if (event.key === 'Enter') {
    window.clearTimeout(debounceTimer)
    commitSearch(draft.value, true)
  }
}

function setSort(value: string) {
  patchQuery({ sort: value === 'title' && !searching.value ? undefined : value })
}

function setLang(value: string) {
  patchQuery({ lang: value === 'any' ? undefined : value || undefined })
}

function setFilterId(key: 'genre' | 'author' | 'series', id: number) {
  patchQuery({ [key]: id ? String(id) : undefined })
}

function clearFilter(key: 'lang' | 'genre' | 'author' | 'series') {
  patchQuery({ [key]: undefined })
}

function clearFilters() {
  void router.replace({ query: q.value ? { q: q.value } : {} })
}

async function openRandom() {
  try {
    const work = await catalog.randomWork()
    if (work?.id) {
      void router.push({ name: 'book', params: { workId: String(work.id) } })
    }
  } catch (err) {
    void err
  }
}

const foundLabel = computed(() => {
  const n = state.value.totalN
  if (n == null) {
    return ''
  }
  if (state.value.capped) {
    return t('catalog.foundCapped', { n: formatCount(n, locale.value) })
  }
  return t('catalog.found', { n: formatCount(n, locale.value) })
})

const emptyText = computed(() => {
  if (searching.value) {
    return t('list.emptySearch')
  }
  if (lang.value || genreFilter.value || authorFilter.value || seriesFilter.value) {
    return t('list.emptyFiltered')
  }
  return t('list.empty')
})

const errorText = computed(() => {
  const err = state.value.error
  if (!err) {
    return t('list.error')
  }
  return errorMessage(err.code, err.params)
})

const hasFilters = computed(() =>
  Boolean(
    lang.value ||
    (!props.genreId && genreFilter.value) ||
    (!props.authorId && authorFilter.value) ||
    (!props.seriesId && seriesFilter.value),
  ),
)

const sortOptions = computed(() => {
  const items = [
    { value: 'title', label: t('catalog.sortTitle') },
    { value: 'added', label: t('catalog.sortAdded') },
  ]
  if (props.seriesOrder) {
    items.unshift({ value: 'seriesno', label: t('catalog.sortSeries') })
  }
  if (searching.value) {
    items.unshift({ value: 'relevance', label: t('catalog.sortRelevance') })
  }
  return items
})

const activeSort = computed(() => {
  if (searching.value) {
    return sort.value || 'relevance'
  }
  if (props.seriesOrder) {
    return sort.value || 'seriesno'
  }
  return sort.value || 'title'
})

const langOptions = computed(() => [
  { value: 'any', label: t('catalog.langAny') },
  { value: 'ru', label: t('catalog.langRu') },
  { value: 'en', label: t('catalog.langEn') },
  { value: 'uk', label: t('catalog.langUk') },
  { value: 'bg', label: t('catalog.langBg') },
])

const langChip = computed(() => langOptions.value.find((item) => item.value === lang.value)?.label)

const genreChip = ref('')
const authorChip = ref('')
const seriesChip = ref('')

watch(
  genreFilter,
  async (id) => {
    if (!id) {
      genreChip.value = ''
      return
    }
    try {
      const item = await catalog.getGenre(id)
      genreChip.value = item?.nameRu ?? ''
    } catch {
      genreChip.value = ''
    }
  },
  { immediate: true },
)

watch(
  authorFilter,
  async (id) => {
    if (!id) {
      authorChip.value = ''
      return
    }
    try {
      const item = await catalog.getAuthor(id)
      authorChip.value = item?.displayName ?? ''
    } catch {
      authorChip.value = ''
    }
  },
  { immediate: true },
)

watch(
  seriesFilter,
  async (id) => {
    if (!id) {
      seriesChip.value = ''
      return
    }
    try {
      const item = await catalog.getSeries(id)
      seriesChip.value = item?.name ?? ''
    } catch {
      seriesChip.value = ''
    }
  },
  { immediate: true },
)

onMounted(() => {
  draft.value = q.value
})
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col gap-3">
    <div class="flex flex-wrap items-center gap-2">
      <label class="sr-only" for="catalog-search">{{ t('catalog.search') }}</label>
      <Input
        id="catalog-search"
        ref="searchInput"
        :model-value="draft"
        :placeholder="t('catalog.search')"
        class="min-w-48 flex-1"
        @update:model-value="onDraftInput"
        @keydown="onSearchKey"
      />
      <Button variant="outline" size="sm" @click="filtersOpen = true">
        <Funnel class="size-4" />
        {{ t('catalog.filters') }}
      </Button>
      <div class="flex items-center gap-2">
        <span class="text-sm text-muted-foreground">{{ t('catalog.sort') }}</span>
        <Select :model-value="activeSort" @update:model-value="(value) => setSort(String(value))">
          <SelectTrigger class="w-48">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="opt in sortOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
      <ToggleGroup
        type="single"
        :model-value="app.catalogView"
        @update:model-value="(value) => value && app.changeCatalogView(value as CatalogView)"
      >
        <ToggleGroupItem value="tile">{{ t('catalog.tile') }}</ToggleGroupItem>
        <ToggleGroupItem value="table">{{ t('catalog.table') }}</ToggleGroupItem>
      </ToggleGroup>
      <Button v-if="showRandom" variant="outline" size="sm" @click="openRandom">
        {{ t('catalog.random') }}
      </Button>
    </div>

    <p v-if="!app.searchIndexReady" class="text-sm text-muted-foreground">
      {{ t('catalog.indexRebuilding') }}
      {{ t('catalog.submitToSearch') }}
    </p>

    <Sheet :open="filtersOpen" @update:open="filtersOpen = $event">
      <SheetContent side="right">
        <SheetHeader>
          <SheetTitle>{{ t('catalog.filters') }}</SheetTitle>
          <SheetDescription>{{ t('catalog.filtersLead') }}</SheetDescription>
        </SheetHeader>
        <div class="grid gap-5">
          <div class="grid gap-2">
            <p class="text-sm text-muted-foreground">{{ t('catalog.lang') }}</p>
            <Select
              :model-value="lang || 'any'"
              @update:model-value="(value) => setLang(String(value))"
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="opt in langOptions" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <EntityFilter
            v-if="!props.genreId"
            kind="genre"
            :model-value="genreFilter"
            @update:model-value="(id) => setFilterId('genre', id)"
          />
          <EntityFilter
            v-if="!props.authorId"
            kind="author"
            :model-value="authorFilter"
            @update:model-value="(id) => setFilterId('author', id)"
          />
          <EntityFilter
            v-if="!props.seriesId"
            kind="series"
            :model-value="seriesFilter"
            @update:model-value="(id) => setFilterId('series', id)"
          />
        </div>
        <SheetFooter>
          <Button variant="outline" @click="clearFilters">{{ t('list.resetFilters') }}</Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>

    <div v-if="hasFilters || q" class="flex flex-wrap items-center gap-2 text-sm">
      <span v-if="foundLabel" class="tabular-nums">{{ foundLabel }}</span>
      <Badge v-if="lang" variant="secondary" class="gap-1 px-3 py-1">
        {{ langChip }}
        <button type="button" :aria-label="t('catalog.removeFilter')" @click="clearFilter('lang')">
          <X class="size-3" />
        </button>
      </Badge>
      <Badge v-if="!props.genreId && genreFilter" variant="secondary" class="gap-1 px-3 py-1">
        {{ genreChip || t('catalog.genre') }}
        <button type="button" :aria-label="t('catalog.removeFilter')" @click="clearFilter('genre')">
          <X class="size-3" />
        </button>
      </Badge>
      <Badge v-if="!props.authorId && authorFilter" variant="secondary" class="gap-1 px-3 py-1">
        {{ authorChip || t('catalog.author') }}
        <button
          type="button"
          :aria-label="t('catalog.removeFilter')"
          @click="clearFilter('author')"
        >
          <X class="size-3" />
        </button>
      </Badge>
      <Badge v-if="!props.seriesId && seriesFilter" variant="secondary" class="gap-1 px-3 py-1">
        {{ seriesChip || t('catalog.series') }}
        <button
          type="button"
          :aria-label="t('catalog.removeFilter')"
          @click="clearFilter('series')"
        >
          <X class="size-3" />
        </button>
      </Badge>
      <Button v-if="hasFilters || q" variant="ghost" size="sm" @click="clearFilters">
        {{ t('list.resetFilters') }}
      </Button>
    </div>
    <p v-else-if="foundLabel" class="text-sm tabular-nums text-muted-foreground">
      {{ foundLabel }}
    </p>

    <ListState
      class="flex min-h-0 flex-1 flex-col"
      :status="state.status"
      :empty-text="emptyText"
      :error-text="errorText"
      @retry="reload"
    >
      <template v-if="hasFilters" #empty>
        <Button class="mt-4" variant="outline" @click="clearFilters">{{
          t('list.resetFilters')
        }}</Button>
      </template>
      <VirtualWorkList
        :items="state.items"
        :view="app.catalogView as CatalogView"
        :query="q"
        :scroll-top="state.scrollTop"
        :loading-more="state.loadingMore"
        :exhausted="state.exhausted"
        :capped="state.capped && searching"
        :total-n="state.totalN"
        @end="loadMore"
        @scroll="catalog.rememberScroll(listKey, $event)"
      />
    </ListState>
  </div>
</template>
