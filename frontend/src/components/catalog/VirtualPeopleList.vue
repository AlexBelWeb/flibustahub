<script setup lang="ts" generic="T extends { id: number }">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useVirtualizer } from '@tanstack/vue-virtual'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  items: T[]
  scrollTop: number
  loadingMore: boolean
  exhausted: boolean
  estimateSize?: number
}>()

const emit = defineEmits<{
  end: []
  scroll: [top: number]
}>()

const { t } = useI18n()
const parentRef = ref<HTMLElement | null>(null)
const size = computed(() => props.estimateSize ?? 56)

const virtualizer = useVirtualizer(
  computed(() => ({
    count: props.items.length,
    getScrollElement: () => parentRef.value,
    estimateSize: () => size.value,
    overscan: 10,
    initialOffset: props.scrollTop,
  })),
)

function measure() {
  virtualizer.value.measure()
}

function onScroll() {
  const el = parentRef.value
  if (!el) {
    return
  }
  emit('scroll', el.scrollTop)
  const last = virtualizer.value.getVirtualItems().at(-1)
  if (last && last.index >= props.items.length - 8) {
    emit('end')
  }
}

onMounted(() => {
  if (parentRef.value && props.scrollTop) {
    parentRef.value.scrollTop = props.scrollTop
  }
  void nextTick(() => measure())
})

onUnmounted(() => {
  if (parentRef.value) {
    emit('scroll', parentRef.value.scrollTop)
  }
})

watch(
  () => props.items.length,
  () => {
    void nextTick(() => measure())
    const last = virtualizer.value.getVirtualItems().at(-1)
    if (last && last.index >= props.items.length - 8) {
      emit('end')
    }
  },
)
</script>

<template>
  <div class="relative min-h-0 flex-1">
    <div ref="parentRef" class="absolute inset-0 overflow-auto" @scroll.passive="onScroll">
      <div class="relative w-full" :style="{ height: `${virtualizer.getTotalSize()}px` }">
        <div
          v-for="row in virtualizer.getVirtualItems()"
          :key="row.key"
          class="absolute top-0 right-0 left-0"
          :style="{ transform: `translateY(${row.start}px)`, height: `${row.size}px` }"
        >
          <slot v-if="items[row.index]" :item="items[row.index]" :index="row.index" />
        </div>
      </div>
      <p v-if="loadingMore" class="p-4 text-center text-sm text-muted-foreground">
        {{ t('list.more') }}
      </p>
      <p
        v-else-if="exhausted && items.length > 0"
        class="p-4 text-center text-sm text-muted-foreground"
      >
        {{ t('list.end') }}
      </p>
    </div>
  </div>
</template>
