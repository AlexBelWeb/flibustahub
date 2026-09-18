<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import HighlightText from '@/components/catalog/HighlightText.vue'
import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { authorParts, coverHue, isBlankTitle } from '@/lib/work'
import type { Work } from '@/types/catalog'

const props = defineProps<{
  work: Work
  query?: string
}>()

const { t } = useI18n()

const title = computed(() =>
  isBlankTitle(props.work.title) ? t('catalog.untitled') : props.work.title,
)
const authors = computed(() => authorParts(props.work.authorsText))
const authorsLine = computed(() => {
  if (authors.value.length <= 2) {
    return authors.value.join(', ')
  }
  return t('catalog.authorsMore', { name: authors.value[0], n: authors.value.length - 1 })
})
const authorsFull = computed(() => authors.value.join(', '))
const hue = computed(() => coverHue(props.work.workKey || String(props.work.id)))
</script>

<template>
  <RouterLink
    :to="{ name: 'book', params: { workId: String(work.id) } }"
    class="flex h-full flex-col overflow-hidden rounded-xl border border-border bg-card text-left outline-none"
    :style="{ '--cover-h': String(hue) }"
  >
    <article class="flex h-full flex-col">
      <div
        class="flex aspect-[2/3] w-full flex-col justify-end p-3 text-primary-foreground"
        :style="{
          background: `hsl(${hue} 32% var(--cover-l, 28%))`,
        }"
      >
        <Tooltip>
          <TooltipTrigger as-child>
            <p class="line-clamp-2 font-display text-sm font-semibold">
              <HighlightText :text="title" :query="query" />
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
      </div>
      <div class="grid gap-1 p-3">
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
          <Badge v-if="work.rating" class="tabular-nums">
            {{ work.rating }}
          </Badge>
          <Badge v-if="work.librate" variant="outline" class="tabular-nums">
            {{ work.librate }}
          </Badge>
        </div>
      </div>
    </article>
  </RouterLink>
</template>
