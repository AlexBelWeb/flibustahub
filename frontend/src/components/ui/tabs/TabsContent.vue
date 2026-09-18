<script setup lang="ts">
import type { TabsContentProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { TabsContent, useForwardProps } from 'reka-ui'
import { cn, reactiveOmit } from '@/lib/utils'

const props = defineProps<TabsContentProps & { class?: HTMLAttributes['class'] }>()
const delegatedProps = reactiveOmit(props as Record<string, unknown>, 'class')
const forwarded = useForwardProps(delegatedProps)
</script>

<template>
  <TabsContent v-bind="forwarded" :class="cn('mt-6 outline-none', props.class)">
    <slot />
  </TabsContent>
</template>
