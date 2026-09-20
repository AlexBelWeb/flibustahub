<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { ChevronLeft } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { useCarousel } from './useCarousel'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()

const { orientation, scrollPrev, canScrollPrev } = useCarousel()
</script>

<template>
  <Button
    type="button"
    variant="outline"
    size="icon"
    :disabled="!canScrollPrev"
    :class="
      cn(
        'absolute size-8 rounded-full',
        orientation === 'horizontal'
          ? 'top-1/2 -left-4 -translate-y-1/2'
          : '-top-4 left-1/2 -translate-x-1/2 rotate-90',
        props.class,
      )
    "
    @click="scrollPrev"
  >
    <ChevronLeft class="size-4" />
    <slot />
  </Button>
</template>
