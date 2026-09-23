<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import BookActions from '@/components/catalog/BookActions.vue'
import BookCover from '@/components/catalog/BookCover.vue'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { formatBytes, languageName } from '@/lib/format'
import { authorParts, coverWash, isBlankTitle, isUnknownAuthor } from '@/lib/work'
import { withWorkQuery } from '@/lib/work-route'
import { useCatalogStore } from '@/stores/catalog'
import type { Author, Work, WorkDetails } from '@/types/catalog'

const props = defineProps<{
  work: Work
  source?: string
}>()

const emit = defineEmits<{
  replace: [work: Work]
}>()

const { t, locale } = useI18n()
const route = useRoute()
const catalog = useCatalogStore()

const details = ref<WorkDetails | null>(null)
const detailsReady = ref(false)
const shuffling = ref(false)
let detailsGen = 0

const title = computed(() =>
  !isBlankTitle(props.work.title) ? props.work.title : t('catalog.untitled'),
)
const wash = computed(() => coverWash(props.work.workKey || String(props.work.id)))
const kicker = computed(() =>
  props.source === 'random' ? t('home.randomBook') : t('home.returnToBook'),
)
const namedAuthors = computed(() =>
  (details.value?.authors ?? []).filter((author) => !isUnknownAuthor(author.displayName)),
)
const authorsFallback = computed(() => {
  const named = authorParts(props.work.authorsText).filter((name) => !isUnknownAuthor(name))
  return named.length ? named.join(', ') : t('catalog.unknownAuthor')
})
const facts = computed(() => {
  const lang = languageName(details.value?.lang || props.work.lang, locale.value)
  const sizeN = details.value?.size ?? props.work.size
  const size = sizeN ? formatBytes(sizeN, locale.value) : ''
  const ext = (details.value?.fileExt || '').replace(/^\./, '').trim()
  const format = ext ? ext.toUpperCase() : ''
  return [lang, size, format].filter(Boolean).join(' · ')
})
const editionId = computed(() => {
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
function authorHref(author: Author) {
  return { name: 'author' as const, params: { authorId: String(author.id) } }
}

async function loadDetails(id: number) {
  const gen = ++detailsGen
  details.value = null
  detailsReady.value = false
  try {
    const next = await window.go.handlers.App.GetWorkDetails(id)
    if (gen !== detailsGen) {
      return
    }
    details.value = next
  } catch {
    if (gen !== detailsGen) {
      return
    }
    details.value = null
  } finally {
    if (gen === detailsGen) {
      detailsReady.value = true
    }
  }
}

async function another() {
  if (shuffling.value) {
    return
  }
  shuffling.value = true
  try {
    for (let i = 0; i < 6; i += 1) {
      const next = await catalog.randomWork()
      if (next?.id && next.id !== props.work.id) {
        emit('replace', next)
        return
      }
      if (next?.id && i === 5) {
        emit('replace', next)
      }
    }
  } finally {
    shuffling.value = false
  }
}

watch(
  () => props.work.id,
  (id) => {
    if (id) {
      void loadDetails(id)
    }
  },
  { immediate: true },
)
</script>

<template>
  <Card class="relative overflow-hidden p-3 sm:p-4 backdrop-panel">
    <div class="pointer-events-none absolute inset-0" :style="{ background: wash }" />
    <div class="relative flex min-w-0 items-center gap-4">
      <div class="aspect-[2/3] h-28 shrink-0 overflow-hidden rounded-xl sm:h-32">
        <BookCover :work="work" prio="open" :observe="false" fit="contain" class="h-full w-full" />
      </div>
      <div class="grid min-w-0 flex-1 gap-1.5">
        <p class="text-sm text-muted-foreground">{{ kicker }}</p>
        <h2 class="type-hero line-clamp-2">{{ title }}</h2>
        <p v-if="namedAuthors.length === 0" class="text-muted-foreground">{{ authorsFallback }}</p>
        <p v-else class="flex flex-wrap gap-x-3 gap-y-1 text-muted-foreground">
          <RouterLink
            v-for="author in namedAuthors"
            :key="author.id"
            :to="authorHref(author)"
            class="underline-offset-4 hover:underline"
          >
            {{ author.displayName }}
          </RouterLink>
        </p>
        <p v-if="facts" class="text-sm text-library tabular-nums">{{ facts }}</p>
        <div class="mt-1 flex flex-wrap items-center gap-2">
          <Button as-child>
            <RouterLink :to="{ query: withWorkQuery(route.query, work.id) }">
              {{ t('home.open') }}
            </RouterLink>
          </Button>
          <BookActions
            v-if="detailsReady"
            hide-download
            :edition-id="editionId"
            :has-file="work.hasFile"
          />
          <Button v-if="source === 'random'" variant="ghost" :disabled="shuffling" @click="another">
            {{ t('home.anotherBook') }}
          </Button>
        </div>
      </div>
    </div>
  </Card>
</template>
