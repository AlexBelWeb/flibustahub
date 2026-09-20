<script setup lang="ts">
import { Star } from '@lucide/vue'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { formatStars, starsFromInternal } from '@/lib/rating'

const props = defineProps<{
  rating?: number | null
  compact?: boolean
}>()

const { t, locale } = useI18n()
const stars = computed(() => starsFromInternal(props.rating))
const label = computed(() => formatStars(props.rating, locale.value))
const filled = computed(() => Math.floor(stars.value))
const half = computed(() => stars.value - filled.value >= 0.5)
</script>

<template>
  <Tooltip v-if="stars > 0" :disabled="compact">
    <TooltipTrigger as-child>
      <span
        class="inline-flex items-center gap-1 text-primary tabular-nums"
        :aria-label="t('personal.ratingValue', { n: label })"
      >
        <span class="inline-flex" aria-hidden="true">
          <Star
            v-for="n in 5"
            :key="n"
            class="size-3.5"
            :class="
              n <= filled
                ? 'fill-primary text-primary'
                : n === filled + 1 && half
                  ? 'fill-primary/50 text-primary'
                  : 'text-muted-foreground'
            "
          />
        </span>
        <span v-if="!compact" class="text-xs">{{ label }}</span>
      </span>
    </TooltipTrigger>
    <TooltipContent>{{ t('personal.ratingValue', { n: label }) }}</TooltipContent>
  </Tooltip>
</template>
