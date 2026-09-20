<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import BookCover from '@/components/catalog/BookCover.vue'
import HighlightText from '@/components/catalog/HighlightText.vue'
import RatingValue from '@/components/catalog/RatingValue.vue'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { formatBytes, languageName } from '@/lib/format'
import { authorParts, isBlankTitle, isUnknownAuthor } from '@/lib/work'
import { withWorkQuery } from '@/lib/work-route'
import type { Work } from '@/types/catalog'

const props = defineProps<{
  work: Work
  query?: string
}>()

const { t, locale } = useI18n()
const route = useRoute()

const title = computed(() =>
  isBlankTitle(props.work.title) ? t('catalog.untitled') : props.work.title,
)
const authors = computed(() => {
  const named = authorParts(props.work.authorsText).filter((name) => !isUnknownAuthor(name))
  return named.length ? named.join(', ') : t('catalog.unknownAuthor')
})
const size = computed(() => (props.work.size ? formatBytes(props.work.size, locale.value) : ''))
const lang = computed(() => languageName(props.work.lang, locale.value))
</script>

<template>
  <RouterLink
    :to="{ query: withWorkQuery(route.query, work.id) }"
    class="grid grid-cols-[2.5rem_minmax(0,2fr)_minmax(0,1.5fr)_minmax(0,1fr)_7rem_6rem_5.5rem] items-center gap-3 border-b border-border px-2 py-2 text-sm hover:bg-accent"
  >
    <BookCover :work="work" compact class="size-10 rounded-md" />
    <Tooltip>
      <TooltipTrigger as-child>
        <span class="truncate font-medium">
          <HighlightText :text="title" :query="query" />
        </span>
      </TooltipTrigger>
      <TooltipContent>{{ title }}</TooltipContent>
    </Tooltip>
    <Tooltip>
      <TooltipTrigger as-child>
        <span class="truncate text-muted-foreground">
          <HighlightText :text="authors" :query="query" />
        </span>
      </TooltipTrigger>
      <TooltipContent>{{ authors }}</TooltipContent>
    </Tooltip>
    <span class="truncate text-muted-foreground">
      {{ work.series }}
      <span v-if="work.seriesNo">{{ t('catalog.seriesNo', { n: work.seriesNo }) }}</span>
    </span>
    <span class="truncate text-muted-foreground">{{ lang }}</span>
    <span class="truncate text-right tabular-nums text-muted-foreground">{{ size }}</span>
    <span class="flex justify-end">
      <RatingValue :rating="work.rating" compact />
    </span>
  </RouterLink>
</template>
