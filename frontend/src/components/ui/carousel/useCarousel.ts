import type { EmblaCarouselType } from 'embla-carousel'
import type { ComputedRef, InjectionKey, Ref } from 'vue'
import { inject } from 'vue'

export interface CarouselContext {
  carouselRef: Ref<HTMLElement | undefined>
  api: Ref<EmblaCarouselType | undefined>
  canScrollPrev: Ref<boolean>
  canScrollNext: Ref<boolean>
  scrollPrev: () => void
  scrollNext: () => void
  orientation: ComputedRef<'horizontal' | 'vertical'>
}

export const CAROUSEL_KEY: InjectionKey<CarouselContext> = Symbol('carousel')

export function useCarousel(): CarouselContext {
  const ctx = inject(CAROUSEL_KEY)
  if (!ctx) {
    throw new Error('useCarousel must be used inside Carousel')
  }
  return ctx
}
