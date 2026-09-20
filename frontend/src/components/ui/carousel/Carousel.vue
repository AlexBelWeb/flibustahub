<script setup lang="ts">
import type { EmblaOptionsType } from 'embla-carousel'
import type { HTMLAttributes } from 'vue'
import emblaCarouselVue from 'embla-carousel-vue'
import { computed, onMounted, onUnmounted, provide, ref, watch } from 'vue'
import { cn } from '@/lib/utils'
import { CAROUSEL_KEY } from './useCarousel'

const props = withDefaults(
  defineProps<{
    opts?: EmblaOptionsType
    orientation?: 'horizontal' | 'vertical'
    class?: HTMLAttributes['class']
  }>(),
  { orientation: 'horizontal' },
)

const options = computed<EmblaOptionsType>(() => ({
  align: 'start',
  skipSnaps: false,
  containScroll: 'trimSnaps',
  axis: props.orientation === 'vertical' ? 'y' : 'x',
  ...props.opts,
}))

const [carouselRef, api] = emblaCarouselVue(options)
const canScrollPrev = ref(false)
const canScrollNext = ref(false)

function onSelect() {
  canScrollPrev.value = api.value?.canScrollPrev() ?? false
  canScrollNext.value = api.value?.canScrollNext() ?? false
}

function scrollPrev() {
  api.value?.scrollPrev()
}

function scrollNext() {
  api.value?.scrollNext()
}

function onKey(event: KeyboardEvent) {
  if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') {
    event.preventDefault()
    scrollPrev()
    return
  }
  if (event.key === 'ArrowRight' || event.key === 'ArrowDown') {
    event.preventDefault()
    scrollNext()
  }
}

onMounted(() => {
  const instance = api.value
  if (!instance) {
    return
  }
  onSelect()
  instance.on('select', onSelect)
  instance.on('reInit', onSelect)
})

watch(api, (instance, prev) => {
  prev?.off('select', onSelect)
  prev?.off('reInit', onSelect)
  if (!instance) {
    return
  }
  onSelect()
  instance.on('select', onSelect)
  instance.on('reInit', onSelect)
})

onUnmounted(() => {
  api.value?.off('select', onSelect)
  api.value?.off('reInit', onSelect)
})

provide(CAROUSEL_KEY, {
  carouselRef,
  api,
  canScrollPrev,
  canScrollNext,
  scrollPrev,
  scrollNext,
  orientation: computed(() => props.orientation ?? 'horizontal'),
})
</script>

<template>
  <div
    :class="cn('relative min-w-0 w-full', props.class)"
    role="region"
    tabindex="0"
    @keydown="onKey"
  >
    <slot />
  </div>
</template>
