<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AlphabetBar from '@/components/catalog/AlphabetBar.vue'
import HighlightText from '@/components/catalog/HighlightText.vue'
import ListState from '@/components/catalog/ListState.vue'
import VirtualPeopleList from '@/components/catalog/VirtualPeopleList.vue'
import { Input } from '@/components/ui/input'
import { errorMessage } from '@/i18n/errors'
import { queryText } from '@/lib/route-query'
import { useCatalogStore } from '@/stores/catalog'
import { useUiStore } from '@/stores/ui'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const catalog = useCatalogStore()
const ui = useUiStore()
const searchInput = ref<{ focus: () => void } | null>(null)

const letter = computed(() => queryText(route.query.letter))
const q = computed(() => queryText(route.query.q).trim())
const listKey = computed(() => JSON.stringify({ letter: letter.value, q: q.value }))
const state = computed(() => catalog.seriesState(listKey.value))
const draft = ref(q.value)

function setLetter(next: string) {
  void router.replace({ query: { ...route.query, letter: next || undefined } })
}

let timer = 0
function onDraft(value: string) {
  draft.value = value
  window.clearTimeout(timer)
  timer = window.setTimeout(() => {
    void router.replace({ query: { ...route.query, q: value.trim() || undefined } })
  }, 250)
}

async function reload() {
  await catalog.loadSeriesList(listKey.value, { letter: letter.value, query: q.value }, true)
}

function loadMore() {
  void catalog.loadSeriesList(listKey.value, { letter: letter.value, query: q.value }, false)
}

watch(
  listKey,
  () => {
    const current = catalog.seriesState(listKey.value)
    if (current.status === 'idle' || current.status === 'error') {
      void reload()
    }
  },
  { immediate: true },
)

watch(
  () => ui.searchFocusNonce,
  () => {
    searchInput.value?.focus()
  },
)

onMounted(() => {
  void catalog.loadAlphabet()
  draft.value = q.value
})

const emptyText = computed(() =>
  q.value ? t('series.emptySearch') : letter.value ? t('list.emptyLetter') : t('list.empty'),
)
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col px-6 py-6">
    <h1 class="mb-4 type-page">{{ t('series.title') }}</h1>
    <Input
      ref="searchInput"
      :model-value="draft"
      :placeholder="t('series.search')"
      class="mb-4 max-w-md"
      @update:model-value="onDraft"
    />
    <AlphabetBar class="mb-4" :letters="catalog.alphabet" :active="letter" @select="setLetter" />
    <ListState
      class="flex min-h-0 flex-1 flex-col"
      :status="state.status"
      :empty-text="emptyText"
      :error-text="
        state.error ? errorMessage(state.error.code, state.error.params) : t('list.error')
      "
      @retry="reload"
    >
      <VirtualPeopleList
        :items="state.items"
        :scroll-top="state.scrollTop"
        :loading-more="state.loadingMore"
        :exhausted="state.exhausted"
        @end="loadMore"
        @scroll="catalog.rememberPeopleScroll('series', listKey, $event)"
      >
        <template #default="{ item }">
          <RouterLink
            :to="{ name: 'seriesDetail', params: { seriesId: String(item.id) } }"
            class="flex h-full items-center justify-between gap-3 border-b border-border px-2 text-sm hover:bg-accent"
          >
            <HighlightText :text="item.name" :query="q" />
            <span class="tabular-nums text-muted-foreground">{{
              t('series.works', item.workCount, { n: item.workCount })
            }}</span>
          </RouterLink>
        </template>
      </VirtualPeopleList>
    </ListState>
  </div>
</template>
