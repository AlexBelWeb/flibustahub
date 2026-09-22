<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { X } from '@lucide/vue'
import BookCover from '@/components/catalog/BookCover.vue'
import BookActions from '@/components/catalog/BookActions.vue'
import ListState from '@/components/catalog/ListState.vue'
import WantButton from '@/components/catalog/WantButton.vue'
import WorkComment from '@/components/catalog/WorkComment.vue'
import WorkRating from '@/components/catalog/WorkRating.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { SheetClose } from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { errorMessage } from '@/i18n/errors'
import { parseBackendError } from '@/lib/backend-error'
import { formatBytes, formatDate, formatFiles, languageName } from '@/lib/format'
import { ratingFromShortcut } from '@/lib/rating'
import { isBlankTitle, isUnknownAuthor, visibleSeriesNo } from '@/lib/work'
import type { ListStatus } from '@/stores/catalog'
import { usePersonalStore } from '@/stores/personal'
import type { Author, WorkDetails, WorkEdition } from '@/types/catalog'

const props = defineProps<{
  workId: number
  drawer?: boolean
}>()

const { t, locale } = useI18n()
const route = useRoute()
const personal = usePersonalStore()

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
const preferredId = computed(() => {
  const work = details.value
  if (!work?.hasFile) {
    return 0
  }
  if (work.preferredEditionId) {
    return work.preferredEditionId
  }
  const preferred = (work.editions ?? []).find((ed) => ed.preferred)
  return preferred?.id || work.editions?.[0]?.id || 0
})
const preferredEdition = computed(() => {
  const work = details.value
  if (!work) {
    return null
  }
  const eds = work.editions ?? []
  return eds.find((ed) => ed.id === preferredId.value) || eds.find((ed) => ed.preferred) || null
})
const size = computed(() => {
  const n = preferredEdition.value?.size ?? details.value?.size
  return n ? formatBytes(n, locale.value) : ''
})
const lang = computed(() => languageName(details.value?.lang, locale.value))
const formatLabel = computed(() => {
  const ext = (preferredEdition.value?.fileExt || details.value?.fileExt || '')
    .replace(/^\./, '')
    .trim()
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
  const parts = [
    ed.fileName,
    ed.archiveName,
    ed.size != null ? formatBytes(ed.size, locale.value) : '',
    ed.addedDate ? formatDate(ed.addedDate, locale.value) : '',
  ]
  return parts.filter(Boolean).join(' · ')
}

function editionExt(ed: WorkEdition) {
  const ext = (ed.fileExt || '').replace(/^\./, '').trim()
  return ext ? ext.toUpperCase() : ''
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

async function onRating(rating: number) {
  if (!details.value) {
    return
  }
  const id = details.value.id
  try {
    await personal.setRating(id, rating)
    if (details.value?.id === id) {
      details.value.rating = rating > 0 ? rating : undefined
    }
  } catch (err) {
    personal.reportSaveError(err, () => {
      void onRating(rating)
    })
  }
}

async function onWant(value: boolean) {
  if (!details.value) {
    return
  }
  const id = details.value.id
  try {
    await personal.setWant(id, value)
    if (details.value?.id === id) {
      details.value.wantToRead = value
    }
  } catch (err) {
    personal.reportSaveError(err, () => {
      void onWant(value)
    })
  }
}

function onShortcut(event: KeyboardEvent) {
  const target = event.target
  if (
    target instanceof HTMLInputElement ||
    target instanceof HTMLTextAreaElement ||
    (target instanceof HTMLElement && target.isContentEditable)
  ) {
    return
  }
  const rating = ratingFromShortcut(event)
  if (rating == null || !details.value) {
    return
  }
  event.preventDefault()
  void onRating(rating)
}

onUnmounted(() => {
  loadGen += 1
  clearAnnTimer()
  window.removeEventListener('keydown', onShortcut)
})

onMounted(() => {
  window.addEventListener('keydown', onShortcut)
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

      <div
        v-if="!drawer"
        class="mx-auto aspect-[2/3] h-auto w-full max-h-80 max-w-[320px] overflow-hidden rounded-xl lg:mx-0 lg:max-h-none"
      >
        <BookCover
          :work="details"
          prio="open"
          :observe="false"
          fit="contain"
          class="h-full w-full"
        />
      </div>

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

        <div v-if="drawer" class="mx-auto aspect-[2/3] w-[58%] shrink-0 overflow-hidden rounded-xl">
          <BookCover
            :work="details"
            prio="open"
            :observe="false"
            fit="contain"
            class="h-full w-full"
          />
        </div>

        <p v-if="details.series" class="text-sm">
          <RouterLink
            v-if="details.seriesId"
            :to="{ name: 'seriesDetail', params: { seriesId: String(details.seriesId) } }"
            class="underline-offset-4 hover:underline"
          >
            {{ details.series }}
            <span v-if="visibleSeriesNo(details.seriesNo)">{{
              t('catalog.seriesNo', { n: visibleSeriesNo(details.seriesNo) })
            }}</span>
          </RouterLink>
          <span v-else>
            {{ details.series }}
            <span v-if="visibleSeriesNo(details.seriesNo)">{{
              t('catalog.seriesNo', { n: visibleSeriesNo(details.seriesNo) })
            }}</span>
          </span>
        </p>

        <p v-if="facts" class="text-sm text-library tabular-nums">{{ facts }}</p>

        <div v-if="details.genres?.length" class="flex flex-wrap gap-2">
          <Badge v-for="genre in details.genres" :key="genre.id" as-child variant="outline">
            <RouterLink :to="{ name: 'genre', params: { genreId: String(genre.id) } }">
              {{ genre.nameRu }}
            </RouterLink>
          </Badge>
        </div>

        <div
          v-if="details.librate || (drawer && details.editionCount > 1) || !details.hasFile"
          class="flex flex-wrap gap-2"
        >
          <Tooltip v-if="details.librate">
            <TooltipTrigger as-child>
              <Badge variant="library" class="px-3 py-1 text-sm tabular-nums" tabindex="0">
                {{ t('catalog.librate', { n: details.librate }) }}
              </Badge>
            </TooltipTrigger>
            <TooltipContent>{{ t('catalog.librateHint') }}</TooltipContent>
          </Tooltip>
          <Badge
            v-if="drawer && details.editionCount > 1"
            variant="library"
            class="px-3 py-1 text-sm tabular-nums"
          >
            {{ filesLabel }}
          </Badge>
          <Badge v-if="!details.hasFile" variant="warning" class="px-3 py-1 text-sm">
            {{ t('catalog.ghost') }}
          </Badge>
        </div>

        <BookActions class="mt-1" :edition-id="preferredId" :has-file="details.hasFile" />

        <div class="flex flex-wrap items-center gap-x-4 gap-y-2">
          <WorkRating :rating="details.rating" @change="onRating" />
          <WantButton :model-value="Boolean(details.wantToRead)" @update:model-value="onWant" />
        </div>

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
            <p class="mb-3 text-destructive">{{ annError || t('book.annotationError') }}</p>
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

        <section v-if="!drawer && (details.editions?.length ?? 0) > 1" class="grid gap-2">
          <h2 class="font-display text-lg font-medium">{{ t('book.editions') }}</h2>
          <ul class="grid gap-2 text-sm text-muted-foreground">
            <li
              v-for="ed in details.editions"
              :key="ed.id"
              :aria-current="ed.preferred ? 'true' : undefined"
              class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4 rounded-lg px-3 py-2"
              :class="ed.preferred ? 'bg-muted/60 text-foreground' : ''"
            >
              <div class="grid min-w-0 gap-1">
                <p v-if="editionExt(ed)" class="text-library">{{ editionExt(ed) }}</p>
                <p class="text-library">{{ editionLine(ed) }}</p>
                <Badge v-if="ed.preferred" variant="secondary" class="w-fit">{{
                  t('book.default')
                }}</Badge>
              </div>
              <BookActions quiet class="shrink-0" :edition-id="ed.id" :has-file="details.hasFile" />
            </li>
          </ul>
        </section>

        <WorkComment :work-id="details.id" :comment="details.comment" />

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
