<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { Badge } from '@/components/ui/badge'
import { useI18n } from 'vue-i18n'
import {
  Carousel,
  CarouselContent,
  CarouselItem,
  CarouselNext,
  CarouselPrevious,
} from '@/components/ui/carousel'

defineProps<{
  tags: Array<{ key: string; to: object; label: string }>
}>()

const { t } = useI18n()
</script>

<template>
  <Carousel
    class="flex min-w-0 items-center gap-4"
    :opts="{ align: 'start', dragFree: true, containScroll: 'trimSnaps' }"
  >
    <CarouselPrevious class="static top-auto left-auto shrink-0 translate-y-0">
      <span class="sr-only">{{ t('home.carouselPrev') }}</span>
    </CarouselPrevious>
    <CarouselContent class="w-auto min-w-0 flex-1">
      <CarouselItem v-for="tag in tags" :key="tag.key" class="basis-auto">
        <Badge
          as-child
          variant="outline"
          class="shrink-0 text-sm whitespace-nowrap hover:bg-accent"
        >
          <RouterLink :to="tag.to">
            {{ tag.label }}
          </RouterLink>
        </Badge>
      </CarouselItem>
    </CarouselContent>
    <CarouselNext class="static top-auto right-auto shrink-0 translate-y-0">
      <span class="sr-only">{{ t('home.carouselNext') }}</span>
    </CarouselNext>
  </Carousel>
</template>
