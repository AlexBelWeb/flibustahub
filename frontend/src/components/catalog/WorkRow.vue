<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import HighlightText from '@/components/catalog/HighlightText.vue'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { formatBytes } from '@/lib/format'
import { authorParts, coverHue, isBlankTitle } from '@/lib/work'
import type { Work } from '@/types/catalog'

const props = defineProps<{
  work: Work
  query?: string
}>()

const { t, locale } = useI18n()

const title = computed(() =>
  isBlankTitle(props.work.title) ? t('catalog.untitled') : props.work.title,
)
const authors = computed(() => authorParts(props.work.authorsText).join(', '))
const hue = computed(() => coverHue(props.work.workKey || String(props.work.id)))
const size = computed(() => (props.work.size ? formatBytes(props.work.size, locale.value) : ''))
</script>

<template>
  <RouterLink
    :to="{ name: 'book', params: { workId: String(work.id) } }"
    class="grid grid-cols-[2.5rem_minmax(0,2fr)_minmax(0,1.5fr)_minmax(0,1fr)_4rem_6rem_4rem] items-center gap-3 border-b border-border px-2 py-2 text-sm hover:bg-accent"
  >
    <span
      class="block size-10 rounded-md"
      :style="{ background: `hsl(${hue} 32% var(--cover-l, 28%))` }"
    />
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
    <span class="truncate text-muted-foreground">{{ work.lang }}</span>
    <span class="truncate text-right tabular-nums text-muted-foreground">{{ size }}</span>
    <span class="truncate text-right tabular-nums">{{ work.rating || work.librate }}</span>
  </RouterLink>
</template>
