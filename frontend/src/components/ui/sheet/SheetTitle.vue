<script setup lang="ts">
import type { DialogTitleProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { DialogTitle } from 'reka-ui'
import { cn, reactiveOmit } from '@/lib/utils'

const props = defineProps<DialogTitleProps & { class?: HTMLAttributes['class'] }>()
const delegatedProps = reactiveOmit(props as Record<string, unknown>, 'class')
</script>

<template>
  <DialogTitle
    v-bind="delegatedProps"
    :class="cn('font-display text-lg font-semibold text-foreground', props.class)"
  >
    <slot />
  </DialogTitle>
</template>
