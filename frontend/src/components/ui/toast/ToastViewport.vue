<script setup lang="ts">
import type { ToastViewportProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { ToastViewport } from 'reka-ui'
import { cn, reactiveOmit } from '@/lib/utils'

const props = defineProps<ToastViewportProps & { class?: HTMLAttributes['class'] }>()
const delegatedProps = reactiveOmit(props as Record<string, unknown>, 'class')
</script>

<template>
  <ToastViewport
    v-bind="delegatedProps"
    :class="
      cn(
        'fixed right-4 bottom-4 z-50 flex w-80 max-w-[calc(100vw-2rem)] flex-col gap-2 outline-none',
        props.class,
      )
    "
  />
</template>
