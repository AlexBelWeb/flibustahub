<script setup lang="ts">
import type { CollapsibleContentProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { CollapsibleContent } from 'reka-ui'
import { cn, reactiveOmit } from '@/lib/utils'

const props = defineProps<CollapsibleContentProps & { class?: HTMLAttributes['class'] }>()
const delegatedProps = reactiveOmit(props as Record<string, unknown>, 'class')
</script>

<template>
  <CollapsibleContent v-bind="delegatedProps" :class="cn('overflow-hidden', props.class)">
    <slot />
  </CollapsibleContent>
</template>
