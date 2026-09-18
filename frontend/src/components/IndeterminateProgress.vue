<script setup lang="ts">
import { ProgressRoot } from 'reka-ui'

defineProps<{
  label: string
}>()
</script>

<template>
  <ProgressRoot
    class="relative h-2 overflow-hidden rounded-full bg-muted"
    :model-value="null"
    :max="100"
    :aria-label="label"
  >
    <div class="indeterminate-bar absolute inset-y-0 left-0 w-1/3 rounded-full bg-primary" />
  </ProgressRoot>
</template>

<style scoped>
.indeterminate-bar {
  animation: indeterminate-bar 1s ease-in-out infinite;
}

@keyframes indeterminate-bar {
  0% {
    transform: translateX(-20%);
    opacity: 0.55;
  }
  50% {
    transform: translateX(180%);
    opacity: 1;
  }
  100% {
    transform: translateX(-20%);
    opacity: 0.55;
  }
}
</style>
