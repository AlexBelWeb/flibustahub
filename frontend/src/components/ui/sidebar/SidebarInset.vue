<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <main :class="cn('flex min-h-0 min-w-0 flex-1 flex-col', props.class)">
    <slot />
  </main>
</template>
