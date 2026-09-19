<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import HighlightText from '@/components/catalog/HighlightText.vue'
import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/ui/command'
import { Skeleton } from '@/components/ui/skeleton'
import { errorMessage } from '@/i18n/errors'
import { isBlankTitle } from '@/lib/work'
import { withWorkQuery } from '@/lib/work-route'
import { useAppStore } from '@/stores/app'
import { useCatalogStore } from '@/stores/catalog'
import { useSearchStore } from '@/stores/search'
import { useUiStore } from '@/stores/ui'
import {
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogOverlay,
  AlertDialogPortal,
  AlertDialogRoot,
  AlertDialogTitle,
  DialogContent,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
} from 'reka-ui'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const app = useAppStore()
const ui = useUiStore()
const search = useSearchStore()
const catalog = useCatalogStore()

const draft = ref('')
const confirmClear = ref(false)
let timer = 0

const open = computed({
  get: () => ui.paletteOpen,
  set: (value: boolean) => {
    ui.paletteOpen = value
  },
})

watch(
  () => ui.paletteOpen,
  (value) => {
    if (value) {
      draft.value = ''
      search.previewSearch('', true)
      void search.loadHistory()
    } else {
      window.clearTimeout(timer)
    }
  },
)

function onInput(value: string) {
  draft.value = value
  window.clearTimeout(timer)
  if (!app.searchIndexReady) {
    return
  }
  timer = window.setTimeout(() => {
    void search.previewSearch(value, true)
  }, 250)
}

function stayOpen(event: Event, fn: () => void) {
  event.preventDefault()
  fn()
}

function closePalette() {
  ui.closePalette()
}

function onComboboxOpen(next: boolean) {
  if (!next) {
    closePalette()
  }
}

function openBooks() {
  const q = draft.value.trim()
  if (q) {
    void search.record(q)
  }
  closePalette()
  void router.push({ name: 'books', query: q ? { q } : {} })
}

function openWork(id: number) {
  const q = draft.value.trim()
  if (q) {
    void search.record(q)
  }
  closePalette()
  void router.push({ query: withWorkQuery(route.query, id) })
}

function openAuthor(id: number) {
  closePalette()
  void router.push({ name: 'author', params: { authorId: String(id) } })
}

function openSeries(id: number) {
  closePalette()
  void router.push({ name: 'seriesDetail', params: { seriesId: String(id) } })
}

function useHistory(item: string) {
  draft.value = item
  void search.previewSearch(item, app.searchIndexReady)
  if (!app.searchIndexReady) {
    void search.record(item)
    closePalette()
    void router.push({ name: 'books', query: { q: item } })
  }
}

async function openRandom() {
  closePalette()
  const work = await catalog.randomWork()
  if (work?.id) {
    void router.push({ query: withWorkQuery(route.query, work.id) })
  }
}

const works = computed(() => search.preview?.works.items ?? [])
const authors = computed(() => search.preview?.authors ?? [])
const series = computed(() => search.preview?.series ?? [])
const idle = computed(() => !draft.value.trim())
</script>

<template>
  <DialogRoot :open="open" @update:open="open = $event">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[80] bg-black/50" />
      <DialogContent
        class="fixed top-[12%] left-1/2 z-[80] w-[min(40rem,calc(100vw-2rem))] -translate-x-1/2 rounded-2xl border border-border bg-card p-0 shadow-lg"
      >
        <DialogTitle class="sr-only">{{ t('palette.title') }}</DialogTitle>
        <Command :ignore-filter="true" :open="true" @update:open="onComboboxOpen">
          <CommandInput
            :model-value="draft"
            :placeholder="t('palette.placeholder')"
            @update:model-value="onInput"
          />
          <p v-if="!app.searchIndexReady" class="px-3 py-2 text-sm text-muted-foreground">
            {{ t('catalog.submitToSearch') }}
          </p>
          <CommandList>
            <template v-if="idle">
              <CommandGroup :heading="t('palette.history')">
                <CommandEmpty v-if="!search.history.length && !search.historyError">
                  {{ t('palette.empty') }}
                </CommandEmpty>
                <p v-if="search.historyError" class="px-2 py-1.5 text-sm">
                  {{ errorMessage(search.historyError.code, search.historyError.params) }}
                </p>
                <CommandItem
                  v-for="(item, index) in search.history"
                  :key="item"
                  :value="'history:' + index + ':' + item"
                  @select="(event) => stayOpen(event, () => useHistory(item))"
                >
                  {{ item }}
                </CommandItem>
              </CommandGroup>
              <CommandSeparator />
              <CommandGroup>
                <CommandItem value="action:random" @select="openRandom">
                  {{ t('catalog.random') }}
                </CommandItem>
                <CommandItem
                  v-if="search.history.length"
                  value="action:clear-history"
                  @select="(event) => stayOpen(event, () => (confirmClear = true))"
                >
                  {{ t('palette.clearHistory') }}
                </CommandItem>
              </CommandGroup>
            </template>
            <template v-else>
              <div
                v-if="search.previewStatus === 'loading'"
                class="grid gap-2 p-2"
                aria-busy="true"
              >
                <Skeleton class="h-8" />
                <Skeleton class="h-8" />
              </div>
              <div v-else-if="search.previewStatus === 'error'" class="p-3">
                <p>
                  {{
                    errorMessage(
                      search.previewError?.code ?? 'internal',
                      search.previewError?.params,
                    )
                  }}
                </p>
                <Button class="mt-2" size="sm" @click="search.previewSearch(draft, true)">{{
                  t('common.retry')
                }}</Button>
              </div>
              <CommandEmpty v-else-if="search.previewStatus === 'empty'">
                {{ t('palette.noResults') }}
              </CommandEmpty>
              <template v-else-if="search.previewStatus === 'ready'">
                <CommandGroup v-if="authors.length" :heading="t('palette.authors')">
                  <CommandItem
                    v-for="item in authors"
                    :key="'a' + item.id"
                    :value="'author:' + item.id"
                    @select="openAuthor(item.id)"
                  >
                    <HighlightText :text="item.displayName" :query="draft" />
                  </CommandItem>
                </CommandGroup>
                <CommandGroup v-if="series.length" :heading="t('palette.series')">
                  <CommandItem
                    v-for="item in series"
                    :key="'s' + item.id"
                    :value="'series:' + item.id"
                    @select="openSeries(item.id)"
                  >
                    <HighlightText :text="item.name" :query="draft" />
                  </CommandItem>
                </CommandGroup>
                <CommandGroup v-if="works.length" :heading="t('palette.books')">
                  <CommandItem
                    v-for="item in works"
                    :key="'w' + item.id"
                    :value="'work:' + item.id"
                    @select="openWork(item.id)"
                  >
                    <HighlightText
                      :text="isBlankTitle(item.title) ? t('catalog.untitled') : item.title"
                      :query="draft"
                    />
                  </CommandItem>
                </CommandGroup>
                <CommandSeparator />
                <CommandGroup>
                  <CommandItem value="action:open-all" @select="openBooks">
                    {{ t('palette.openAll') }}
                  </CommandItem>
                </CommandGroup>
              </template>
            </template>
          </CommandList>
          <div class="flex justify-end border-t border-border p-2">
            <Button variant="ghost" size="sm" @click="closePalette()">{{
              t('common.cancel')
            }}</Button>
          </div>
        </Command>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>

  <AlertDialogRoot :open="confirmClear" @update:open="confirmClear = $event">
    <AlertDialogPortal>
      <AlertDialogOverlay class="fixed inset-0 z-[90] bg-black/50" />
      <AlertDialogContent
        class="fixed top-1/2 left-1/2 z-[90] w-[min(28rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-border bg-card p-6"
      >
        <AlertDialogTitle class="font-display text-lg">{{
          t('palette.clearHistoryTitle')
        }}</AlertDialogTitle>
        <AlertDialogDescription class="mt-2 text-sm text-muted-foreground">
          {{ t('palette.clearHistoryBody') }}
        </AlertDialogDescription>
        <div class="mt-6 flex justify-end gap-2">
          <AlertDialogCancel as-child>
            <Button variant="outline">{{ t('common.dismiss') }}</Button>
          </AlertDialogCancel>
          <AlertDialogAction as-child>
            <Button
              @click="
                () => {
                  confirmClear = false
                  void search.clearHistory()
                }
              "
            >
              {{ t('palette.clearHistory') }}
            </Button>
          </AlertDialogAction>
        </div>
      </AlertDialogContent>
    </AlertDialogPortal>
  </AlertDialogRoot>
</template>
