<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import ListState from '@/components/catalog/ListState.vue'
import { Carousel, CarouselContent, CarouselNext, CarouselPrevious } from '@/components/ui/carousel'

defineProps<{
  title: string
  moreLabel: string
  moreTo: object
  status: string
  emptyText: string
  errorText: string
}>()

const emit = defineEmits<{
  retry: []
}>()

const { t } = useI18n()
</script>

<template>
  <section class="grid min-w-0 gap-4">
    <div class="flex flex-wrap items-baseline justify-between gap-3">
      <h2 class="type-section">{{ title }}</h2>
      <RouterLink :to="moreTo" class="text-sm underline-offset-4 hover:underline">
        {{ moreLabel }}
      </RouterLink>
    </div>
    <ListState
      class="flex-none"
      :status="status"
      :empty-text="emptyText"
      :error-text="errorText"
      @retry="emit('retry')"
    >
      <Carousel class="flex min-w-0 items-center gap-4">
        <CarouselPrevious class="static top-auto left-auto shrink-0 translate-y-0">
          <span class="sr-only">{{ t('home.carouselPrev') }}</span>
        </CarouselPrevious>
        <CarouselContent class="w-auto min-w-0 flex-1 py-6">
          <slot />
        </CarouselContent>
        <CarouselNext class="static top-auto right-auto shrink-0 translate-y-0">
          <span class="sr-only">{{ t('home.carouselNext') }}</span>
        </CarouselNext>
      </Carousel>
    </ListState>
  </section>
</template>
