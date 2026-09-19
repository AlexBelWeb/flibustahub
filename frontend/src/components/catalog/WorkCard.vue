<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import BookCover from '@/components/catalog/BookCover.vue'
import HighlightText from '@/components/catalog/HighlightText.vue'
import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { authorParts, isBlankTitle, isUnknownAuthor } from '@/lib/work'
import { withWorkQuery } from '@/lib/work-route'
import type { Work } from '@/types/catalog'

const props = defineProps<{
  work: Work
  query?: string
}>()

const { t } = useI18n()
const route = useRoute()

const title = computed(() =>
  isBlankTitle(props.work.title) ? t('catalog.untitled') : props.work.title,
)
const authors = computed(() =>
  authorParts(props.work.authorsText).filter((name) => !isUnknownAuthor(name)),
)
const authorsLine = computed(() => {
  if (authors.value.length === 0) {
    return t('catalog.unknownAuthor')
  }
  if (authors.value.length <= 2) {
    return authors.value.join(', ')
  }
  return t('catalog.authorsMore', { name: authors.value[0], n: authors.value.length - 1 })
})
const authorsFull = computed(() =>
  authors.value.length ? authors.value.join(', ') : t('catalog.unknownAuthor'),
)
</script>

<template>
  <RouterLink
    :to="{ query: withWorkQuery(route.query, work.id) }"
    class="flex h-full flex-col overflow-hidden rounded-xl border border-border bg-card text-left outline-none"
  >
    <article class="flex h-full flex-col">
      <BookCover :work="work" class="aspect-[2/3] w-full">
        <template #plate="{ title: plateTitle, authors: plateAuthors }">
          <Tooltip>
            <TooltipTrigger as-child>
              <p class="line-clamp-2 font-display text-sm font-semibold">
                <HighlightText :text="plateTitle" :query="query" />
              </p>
            </TooltipTrigger>
            <TooltipContent>{{ title }}</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger as-child>
              <p class="mt-1 line-clamp-1 text-xs opacity-90">
                <HighlightText :text="authorsLine" :query="query" />
              </p>
            </TooltipTrigger>
            <TooltipContent>{{ authorsFull }}</TooltipContent>
          </Tooltip>
          <span class="sr-only">{{ plateAuthors }}</span>
        </template>
      </BookCover>
      <div class="grid gap-1 p-3">
        <Tooltip>
          <TooltipTrigger as-child>
            <p class="line-clamp-2 font-medium">
              <HighlightText :text="title" :query="query" />
            </p>
          </TooltipTrigger>
          <TooltipContent>{{ title }}</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger as-child>
            <p class="line-clamp-1 text-sm text-muted-foreground">
              <HighlightText :text="authorsLine" :query="query" />
            </p>
          </TooltipTrigger>
          <TooltipContent>{{ authorsFull }}</TooltipContent>
        </Tooltip>
        <p v-if="work.series" class="line-clamp-1 text-xs text-muted-foreground">
          {{ work.series }}
          <span v-if="work.seriesNo">{{ t('catalog.seriesNo', { n: work.seriesNo }) }}</span>
        </p>
        <div class="flex flex-wrap gap-1">
          <Badge v-if="work.editionCount > 1" variant="secondary" class="tabular-nums">
            {{ t('catalog.files', work.editionCount, { n: work.editionCount }) }}
          </Badge>
          <Badge v-if="!work.hasFile" variant="muted">
            {{ t('catalog.ghost') }}
          </Badge>
        </div>
      </div>
    </article>
  </RouterLink>
</template>
