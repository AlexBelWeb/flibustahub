<script setup lang="ts">
import type { TabsListProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { TabsList, useForwardProps } from 'reka-ui'
import { cn, reactiveOmit } from '@/lib/utils'

const props = defineProps<TabsListProps & { class?: HTMLAttributes['class'] }>()
const delegatedProps = reactiveOmit(props as Record<string, unknown>, 'class')
const forwarded = useForwardProps(delegatedProps)
</script>

<template>
  <TabsList v-bind="forwarded" :class="cn('inline-flex flex-wrap items-center gap-2', props.class)">
    <slot />
  </TabsList>
</template>
