<script setup lang="ts">
import { computed } from 'vue'
import { ProgressIndicator, ProgressRoot } from 'reka-ui'

const props = withDefaults(
  defineProps<{
    modelValue?: number | null
    max?: number
    label: string
  }>(),
  { modelValue: null, max: 100 },
)

const offset = computed(() => {
  const value = props.modelValue
  if (value == null || props.max <= 0) {
    return 100
  }
  const ratio = Math.min(1, Math.max(0, value / props.max))
  return 100 - ratio * 100
})
</script>

<template>
  <ProgressRoot
    class="ui-progress relative h-2 overflow-hidden rounded-full bg-muted"
    :model-value="modelValue"
    :max="max"
    :get-value-label="() => label"
  >
    <ProgressIndicator
      v-if="modelValue == null"
      class="absolute inset-y-0 left-0 w-1/3 rounded-full bg-primary"
    />
    <ProgressIndicator
      v-else
      class="motion-base h-full w-full bg-primary transition-transform"
      :style="{ transform: `translateX(-${offset}%)` }"
    />
  </ProgressRoot>
</template>
