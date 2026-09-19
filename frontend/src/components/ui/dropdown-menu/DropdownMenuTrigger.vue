<script setup lang="ts">
import type { DropdownMenuTriggerProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { DropdownMenuTrigger, useForwardProps } from 'reka-ui'
import { cn, reactiveOmit } from '@/lib/utils'

const props = defineProps<DropdownMenuTriggerProps & { class?: HTMLAttributes['class'] }>()
const delegatedProps = reactiveOmit(props as Record<string, unknown>, 'class')
const forwardedProps = useForwardProps(delegatedProps)
</script>

<template>
  <DropdownMenuTrigger v-bind="forwardedProps" :class="cn(props.class)">
    <slot />
  </DropdownMenuTrigger>
</template>
