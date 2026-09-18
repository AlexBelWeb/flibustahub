<script setup lang="ts">
import type { ToastProviderProps } from 'reka-ui'
import { ToastProvider } from 'reka-ui'

const props = withDefaults(defineProps<ToastProviderProps>(), {
  duration: 4000,
  swipeDirection: 'right',
})
</script>

<template>
  <ToastProvider v-bind="props">
    <slot />
  </ToastProvider>
</template>
