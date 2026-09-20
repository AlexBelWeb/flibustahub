<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { ChevronRight } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { useCarousel } from './useCarousel'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()

const { orientation, scrollNext, canScrollNext } = useCarousel()
</script>

<template>
  <Button
    type="button"
    variant="outline"
    size="icon"
    :disabled="!canScrollNext"
    :class="
      cn(
        'absolute size-8 rounded-full',
        orientation === 'horizontal'
          ? 'top-1/2 -right-4 -translate-y-1/2'
          : '-bottom-4 left-1/2 -translate-x-1/2 rotate-90',
        props.class,
      )
    "
    @click="scrollNext"
  >
    <ChevronRight class="size-4" />
    <slot />
  </Button>
</template>
