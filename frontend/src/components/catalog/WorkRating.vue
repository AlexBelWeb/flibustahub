<script setup lang="ts">
import { Star } from '@lucide/vue'
import { RatingItem, RatingItemIndicator, RatingRoot } from 'reka-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { internalFromStars, starsFromInternal } from '@/lib/rating'

const props = withDefaults(
  defineProps<{
    rating?: number | null
    disabled?: boolean
  }>(),
  { disabled: false },
)

const emit = defineEmits<{
  change: [rating: number]
}>()

const { t } = useI18n()

const stars = computed(() => starsFromInternal(props.rating))

function onStars(value: number | null | undefined) {
  emit('change', internalFromStars(value ?? 0))
}

function onKey(event: KeyboardEvent) {
  if (props.disabled) {
    return
  }
  if (event.key === 'Delete' || event.key === 'Backspace') {
    if (stars.value > 0) {
      event.preventDefault()
      emit('change', 0)
    }
  }
}
</script>

<template>
  <div class="flex flex-wrap items-center gap-2" @keydown="onKey">
    <Tooltip>
      <TooltipTrigger as-child>
        <RatingRoot
          :model-value="stars"
          :length="5"
          :step="0.5"
          :disabled="disabled"
          clearable
          hoverable
          class="flex gap-0.5"
          :aria-label="t('personal.rating')"
          @update:model-value="onStars"
        >
          <template #default="{ items }">
            <RatingItem
              v-for="item in items"
              :key="item"
              :item="item"
              class="relative inline-flex size-6"
            >
              <template #default="{ steps }">
                <Star class="size-6 text-muted-foreground" aria-hidden="true" />
                <RatingItemIndicator
                  v-for="step in steps"
                  :key="step"
                  :step="step"
                  class="absolute inset-y-0 left-0 overflow-hidden opacity-0 data-[state=active]:opacity-100"
                  :style="{
                    width: 'var(--reka-rating-item-step-width)',
                    zIndex: 'var(--reka-rating-item-step-z-index)',
                  }"
                >
                  <Star class="size-6 fill-primary text-primary" aria-hidden="true" />
                </RatingItemIndicator>
              </template>
            </RatingItem>
          </template>
        </RatingRoot>
      </TooltipTrigger>
      <TooltipContent>{{ t('personal.ratingShortcuts') }}</TooltipContent>
    </Tooltip>
    <Button
      v-if="!disabled && stars > 0"
      type="button"
      variant="ghost"
      size="sm"
      @click="emit('change', 0)"
    >
      {{ t('personal.clearRating') }}
    </Button>
  </div>
</template>
