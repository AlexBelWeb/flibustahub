<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { X } from '@lucide/vue'
import BookCover from '@/components/catalog/BookCover.vue'
import ListState from '@/components/catalog/ListState.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { SheetClose } from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { errorMessage } from '@/i18n/errors'
import { parseBackendError } from '@/lib/backend-error'
import { formatBytes, formatFiles, languageName } from '@/lib/format'
import { isBlankTitle, isUnknownAuthor } from '@/lib/work'
import type { ListStatus } from '@/stores/catalog'
import type { Author, WorkDetails, WorkEdition } from '@/types/catalog'

const props = defineProps<{
  workId: number
  drawer?: boolean
}>()

const { t, locale } = useI18n()
const route = useRoute()

const status = ref<ListStatus>('loading')
const details = ref<WorkDetails | null>(null)
const error = ref('')
const annStatus = ref<ListStatus>('idle')
const annotation = ref('')
const annError = ref('')
const annSlow = ref(false)
const bodyRef = ref<HTMLElement | null>(null)
let annTimer: ReturnType<typeof setTimeout> | null = null
let loadGen = 0

const title = computed(() =>
  details.value && !isBlankTitle(details.value.title) ? details.value.title : t('catalog.untitled'),
)
const size = computed(() =>
  details.value?.size ? formatBytes(details.value.size, locale.value) : '',
)
const lang = computed(() => languageName(details.value?.lang, locale.value))
const formatLabel = computed(() => {
  const ext = (details.value?.fileExt || '').replace(/^\./, '').trim()
  return ext ? ext.toUpperCase() : ''
})
const facts = computed(() =>
  [lang.value, size.value, formatLabel.value].filter(Boolean).join(' · '),
)
const namedAuthors = computed(() =>
  (details.value?.authors ?? []).filter((author) => !isUnknownAuthor(author.displayName)),
)
const filesLabel = computed(() =>
  details.value ? formatFiles(details.value.editionCount, locale.value) : '',
)

function authorHref(author: Author) {
  return { name: 'author' as const, params: { authorId: String(author.id) } }
}

function editionLine(ed: WorkEdition) {
  const ext = (ed.fileExt || '').replace(/^\./, '').trim()
  const parts = [
    ed.archiveName,
    ed.size != null ? formatBytes(ed.size, locale.value) : '',
    ext ? ext.toUpperCase() : '',
  ]
  return parts.filter(Boolean).join(' · ')
}

function clearAnnTimer() {
  if (annTimer) {
    clearTimeout(annTimer)
    annTimer = null
  }
  annSlow.value = false
}

async function loadAnnotation(work: WorkDetails, gen: number) {
  if (work.annotation) {
    annotation.value = work.annotation
    annStatus.value = 'ready'
    return
  }
  if (work.annotationChecked) {
    annotation.value = ''
    annStatus.value = 'empty'
    return
  }
  annStatus.value = 'loading'
  annError.value = ''
  clearAnnTimer()
  annTimer = setTimeout(() => {
    annSlow.value = true
  }, 1000)
  try {
    const got = await window.go.handlers.App.GetAnnotation(work.id)
    if (gen !== loadGen) {
      return
    }
    annotation.value = got.text ?? ''
    annStatus.value = annotation.value ? 'ready' : 'empty'
  } catch (err) {
    if (gen !== loadGen) {
      return
    }
    const be = parseBackendError(err)
    annError.value = errorMessage(be.code, be.params)
    annStatus.value = 'error'
  } finally {
    if (gen === loadGen) {
      clearAnnTimer()
    }
  }
}

async function load() {
  const gen = ++loadGen
  if (!props.workId) {
    status.value = 'missing'
    details.value = null
    return
  }
  status.value = 'loading'
  error.value = ''
  annotation.value = ''
  annStatus.value = 'idle'
  clearAnnTimer()
  try {
    const next = await window.go.handlers.App.GetWorkDetails(props.workId)
    if (gen !== loadGen) {
      return
    }
    details.value = next
    status.value = next?.id ? 'ready' : 'missing'
    if (next?.id) {
      void loadAnnotation(next, gen)
    }
  } catch (err) {
    if (gen !== loadGen) {
      return
    }
    const be = parseBackendError(err)
    if (be.code === 'not_found') {
      status.value = 'missing'
      details.value = null
      return
    }
    error.value = errorMessage(be.code, be.params)
    status.value = 'error'
  }
}

onUnmounted(() => {
  loadGen += 1
  clearAnnTimer()
})

watch(
  () => props.workId,
  async () => {
    await load()
    await nextTick()
    if (bodyRef.value) {
      bodyRef.value.scrollTop = 0
    }
  },
  { immediate: true },
)
</script>

<template>
  <ListState
    :status="status"
    :empty-text="t('common.notFound')"
    :error-text="error || t('list.error')"
    class="flex min-h-0 flex-1 flex-col"
    @retry="load"
  >
    <article
      v-if="details"
      class="flex min-h-0 flex-1 flex-col"
      :class="
        drawer
          ? ''
          : 'gap-6 lg:grid lg:grid-cols-[minmax(280px,320px)_minmax(0,1fr)] lg:items-start lg:content-start'
      "
    >
      <header
        v-if="drawer"
        class="flex shrink-0 items-start gap-4 border-b border-border bg-background px-6 py-4"
      >
        <div class="grid min-w-0 flex-1 gap-2">
          <h1 class="font-display text-2xl font-semibold">{{ title }}</h1>
          <p v-if="namedAuthors.length === 0" class="text-muted-foreground">
            {{ t('catalog.unknownAuthor') }}
          </p>
          <p v-else class="flex flex-wrap gap-x-3 gap-y-1">
            <RouterLink
              v-for="author in namedAuthors"
              :key="author.id"
              :to="authorHref(author)"
              class="underline-offset-4 hover:underline"
            >
              {{ author.displayName }}
            </RouterLink>
          </p>
        </div>
        <SheetClose
          class="mt-1 shrink-0 rounded-sm opacity-70 hover:opacity-100"
          :aria-label="t('common.dismiss')"
        >
          <X class="size-4" />
        </SheetClose>
      </header>

      <BookCover
        v-if="!drawer"
        :work="details"
        prio="open"
        :observe="false"
        fit="contain"
        class="mx-auto aspect-[2/3] h-auto w-full max-h-80 max-w-[320px] rounded-xl lg:mx-0 lg:max-h-none"
      />

      <div
        ref="bodyRef"
        class="min-w-0"
        :class="
          drawer
            ? 'flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-6 py-4 pb-8'
            : 'grid gap-6 content-start'
        "
      >
        <header v-if="!drawer" class="grid gap-2">
          <h1 class="font-display text-3xl font-semibold sm:text-4xl">{{ title }}</h1>
          <p v-if="namedAuthors.length === 0" class="text-lg text-muted-foreground">
            {{ t('catalog.unknownAuthor') }}
          </p>
          <p v-else class="flex flex-wrap gap-x-3 gap-y-1 text-lg">
            <RouterLink
              v-for="author in namedAuthors"
              :key="author.id"
              :to="authorHref(author)"
              class="underline-offset-4 hover:underline"
            >
              {{ author.displayName }}
            </RouterLink>
          </p>
        </header>

        <div v-if="drawer" class="mx-auto w-[58%] shrink-0">
          <BookCover
            :work="details"
            prio="open"
            :observe="false"
            fit="contain"
            class="aspect-[2/3] w-full overflow-hidden rounded-xl"
          />
        </div>

        <p v-if="facts" class="text-sm text-muted-foreground tabular-nums">{{ facts }}</p>

        <div v-if="details.series || details.genres?.length" class="flex flex-wrap gap-2">
          <RouterLink
            v-if="details.series && details.seriesId"
            :to="{ name: 'seriesDetail', params: { seriesId: String(details.seriesId) } }"
            class="rounded-full border border-border px-3 py-1 text-sm"
          >
            {{ details.series }}
            <span v-if="details.seriesNo">{{
              t('catalog.seriesNo', { n: details.seriesNo })
            }}</span>
          </RouterLink>
          <span
            v-else-if="details.series"
            class="rounded-full border border-border px-3 py-1 text-sm"
          >
            {{ details.series }}
            <span v-if="details.seriesNo">{{
              t('catalog.seriesNo', { n: details.seriesNo })
            }}</span>
          </span>
          <RouterLink
            v-for="genre in details.genres"
            :key="genre.id"
            :to="{ name: 'genre', params: { genreId: String(genre.id) } }"
            class="rounded-full border border-border px-3 py-1 text-sm"
          >
            {{ genre.nameRu }}
          </RouterLink>
        </div>

        <div v-if="details.editionCount > 1 || !details.hasFile" class="flex flex-wrap gap-2">
          <Badge
            v-if="details.editionCount > 1"
            variant="secondary"
            class="px-3 py-1 text-sm tabular-nums"
          >
            {{ filesLabel }}
          </Badge>
          <Badge v-if="!details.hasFile" variant="muted" class="px-3 py-1 text-sm">
            {{ t('catalog.ghost') }}
          </Badge>
        </div>

        <section v-if="!drawer && details.editions?.length" class="grid gap-2">
          <h2 class="font-display text-lg font-medium">{{ t('book.editions') }}</h2>
          <ul class="grid gap-1 text-sm text-muted-foreground">
            <li v-for="(ed, index) in details.editions" :key="`${ed.archiveName}-${index}`">
              {{ editionLine(ed) }}
            </li>
          </ul>
        </section>

        <section class="grid gap-2">
          <h2 class="font-display text-lg font-medium">{{ t('book.annotation') }}</h2>
          <div v-if="annStatus === 'loading' && !annSlow" class="grid gap-2" aria-busy="true">
            <span class="sr-only">{{ t('common.loading') }}</span>
            <Skeleton class="h-4 w-full" />
            <Skeleton class="h-4 w-5/6" />
            <Skeleton class="h-4 w-2/3" />
          </div>
          <p v-else-if="annStatus === 'loading' && annSlow" class="text-sm text-muted-foreground">
            {{ t('book.readingDisk') }}
          </p>
          <div v-else-if="annStatus === 'error'" class="rounded-xl border border-border p-4">
            <p class="mb-3">{{ annError || t('book.annotationError') }}</p>
            <Button
              variant="outline"
              size="sm"
              @click="details && loadAnnotation(details, loadGen)"
            >
              {{ t('common.retry') }}
            </Button>
          </div>
          <p v-else-if="annStatus === 'empty'" class="text-muted-foreground">
            {{ t('book.annotationEmpty') }}
          </p>
          <p v-else class="whitespace-pre-wrap text-sm leading-relaxed">{{ annotation }}</p>
        </section>

        <nav v-if="details.prevWorkId || details.nextWorkId" class="flex gap-2">
          <Button v-if="details.prevWorkId" variant="outline" as-child>
            <RouterLink
              :to="
                drawer
                  ? { query: { ...route.query, work: String(details.prevWorkId) } }
                  : { name: 'book', params: { workId: String(details.prevWorkId) } }
              "
            >
              {{ t('book.prev') }}
            </RouterLink>
          </Button>
          <Button v-if="details.nextWorkId" variant="outline" as-child>
            <RouterLink
              :to="
                drawer
                  ? { query: { ...route.query, work: String(details.nextWorkId) } }
                  : { name: 'book', params: { workId: String(details.nextWorkId) } }
              "
            >
              {{ t('book.next') }}
            </RouterLink>
          </Button>
        </nav>

        <p v-if="drawer">
          <Button variant="outline" as-child>
            <RouterLink :to="{ name: 'book', params: { workId: String(details.id) } }" replace>
              {{ t('book.fullscreen') }}
            </RouterLink>
          </Button>
        </p>
      </div>
    </article>
  </ListState>
</template>
