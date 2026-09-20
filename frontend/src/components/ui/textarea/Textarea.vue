<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { cn } from '@/lib/utils'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  class?: HTMLAttributes['class']
  modelValue?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const el = ref<HTMLTextAreaElement | null>(null)
const classes = computed(() =>
  cn(
    'flex min-h-24 w-full resize-none rounded-lg border border-input bg-background px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:ring-1 focus-visible:ring-ring focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50',
    props.class,
  ),
)

function resize() {
  const node = el.value
  if (!node) {
    return
  }
  node.style.height = 'auto'
  node.style.height = `${node.scrollHeight}px`
}

function onInput(event: Event) {
  emit('update:modelValue', (event.target as HTMLTextAreaElement).value)
  void nextTick(resize)
}

watch(
  () => props.modelValue,
  () => {
    void nextTick(resize)
  },
)

onMounted(() => {
  resize()
})

function focus() {
  el.value?.focus()
}

function restoreCursor(start: number, end: number) {
  const node = el.value
  if (!node) {
    return
  }
  node.focus()
  node.setSelectionRange(start, end)
}

defineExpose({ focus, el, restoreCursor, resize })
</script>

<template>
  <textarea ref="el" :class="classes" :value="modelValue" v-bind="$attrs" @input="onInput" />
</template>
