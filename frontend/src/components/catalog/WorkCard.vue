<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import BookCover from '@/components/catalog/BookCover.vue'
import HighlightText from '@/components/catalog/HighlightText.vue'
import RatingValue from '@/components/catalog/RatingValue.vue'
import WantButton from '@/components/catalog/WantButton.vue'
import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { formatFiles } from '@/lib/format'
import { cn } from '@/lib/utils'
import { authorParts, isBlankTitle, isUnknownAuthor } from '@/lib/work'
import { withWorkQuery } from '@/lib/work-route'
import { usePersonalStore } from '@/stores/personal'
import type { Work } from '@/types/catalog'

const props = defineProps<{
  work: Work
  query?: string
  class?: HTMLAttributes['class']
}>()

const { t, locale } = useI18n()
const route = useRoute()
const personal = usePersonalStore()

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

async function onWant(value: boolean) {
  try {
    await personal.setWant(props.work.id, value)
  } catch (err) {
    personal.reportSaveError(err, () => {
      void onWant(value)
    })
  }
}
</script>

<template>
  <article
    :class="
      cn('elevate flex h-full flex-col overflow-hidden rounded-xl bg-card text-left', props.class)
    "
  >
    <RouterLink
      :to="{ query: withWorkQuery(route.query, work.id) }"
      class="flex min-h-0 flex-1 flex-col outline-none"
    >
      <BookCover :work="work" class="aspect-[2/3] w-full" />
      <div class="grid gap-1 p-3">
        <Tooltip>
          <TooltipTrigger as-child>
            <p class="line-clamp-2 text-sm leading-5 font-medium">
              <HighlightText :text="title" :query="query" />
            </p>
          </TooltipTrigger>
          <TooltipContent>{{ title }}</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger as-child>
            <p class="truncate text-xs leading-4 text-muted-foreground">
              <HighlightText :text="authorsLine" :query="query" />
            </p>
          </TooltipTrigger>
          <TooltipContent>{{ authorsFull }}</TooltipContent>
        </Tooltip>
        <p v-if="work.series" class="truncate text-xs leading-4 text-muted-foreground">
          {{ work.series }}
          <span v-if="work.seriesNo">{{ t('catalog.seriesNo', { n: work.seriesNo }) }}</span>
        </p>
        <div class="flex flex-wrap items-center gap-1">
          <RatingValue :rating="work.rating" compact />
          <Badge v-if="work.editionCount > 1" variant="library" class="tabular-nums">
            {{ formatFiles(work.editionCount, locale) }}
          </Badge>
          <Badge v-if="!work.hasFile" variant="warning">
            {{ t('catalog.ghost') }}
          </Badge>
        </div>
      </div>
    </RouterLink>
    <div class="px-3 pb-3" @click.stop @pointerdown.stop @keydown.stop>
      <WantButton compact :model-value="Boolean(work.wantToRead)" @update:model-value="onWant" />
    </div>
  </article>
</template>
