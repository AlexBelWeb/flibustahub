<script setup lang="ts">
import type { ToastRootEmits, ToastRootProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { ToastRoot, useForwardPropsEmits } from 'reka-ui'
import { cn, reactiveOmit } from '@/lib/utils'

const props = defineProps<ToastRootProps & { class?: HTMLAttributes['class'] }>()
const emits = defineEmits<ToastRootEmits>()
const delegatedProps = reactiveOmit(props as Record<string, unknown>, 'class')
const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <ToastRoot
    v-bind="forwarded"
    :class="
      cn(
        'motion-pop elevate-raised pointer-events-auto relative rounded-xl bg-card px-4 py-3 text-sm text-card-foreground',
        props.class,
      )
    "
  >
    <slot />
  </ToastRoot>
</template>
