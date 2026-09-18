<script setup lang="ts">
import type { ToastDescriptionProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { ToastDescription } from 'reka-ui'
import { cn, reactiveOmit } from '@/lib/utils'

const props = defineProps<ToastDescriptionProps & { class?: HTMLAttributes['class'] }>()
const delegatedProps = reactiveOmit(props as Record<string, unknown>, 'class')
</script>

<template>
  <ToastDescription v-bind="delegatedProps" :class="cn('min-w-0 flex-1', props.class)">
    <slot />
  </ToastDescription>
</template>
