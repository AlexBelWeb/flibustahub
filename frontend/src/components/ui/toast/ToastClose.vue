<script setup lang="ts">
import type { ToastCloseProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { ToastClose } from 'reka-ui'
import { cn, reactiveOmit } from '@/lib/utils'

const props = defineProps<ToastCloseProps & { class?: HTMLAttributes['class'] }>()
const delegatedProps = reactiveOmit(props as Record<string, unknown>, 'class')
</script>

<template>
  <ToastClose v-bind="delegatedProps" :class="cn(props.class)">
    <slot />
  </ToastClose>
</template>
