<script setup lang="ts">
import { highlightChunks } from '@/lib/highlight'

defineProps<{
  text: string
  query?: string
}>()
</script>

<template>
  <span>
    <template v-if="!query">{{ text }}</template>
    <template v-else>
      <span
        v-for="(chunk, i) in highlightChunks(text, query)"
        :key="i"
        :class="chunk.match ? 'bg-primary/25' : ''"
        >{{ chunk.text }}</span
      >
    </template>
  </span>
</template>
