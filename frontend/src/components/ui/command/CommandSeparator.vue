<script setup lang="ts">
import type { SeparatorProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { ComboboxSeparator } from 'reka-ui'
import { cn, reactiveOmit } from '@/lib/utils'

const props = defineProps<SeparatorProps & { class?: HTMLAttributes['class'] }>()
const delegatedProps = reactiveOmit(props as Record<string, unknown>, 'class')
</script>

<template>
  <ComboboxSeparator v-bind="delegatedProps" :class="cn('-mx-1 h-px bg-border', props.class)" />
</template>
