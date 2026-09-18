<script setup lang="ts">
import type { TabsTriggerProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { TabsTrigger, useForwardProps } from 'reka-ui'
import { cn, reactiveOmit } from '@/lib/utils'
import { buttonVariants } from '@/components/ui/button'

const props = defineProps<TabsTriggerProps & { class?: HTMLAttributes['class'] }>()
const delegatedProps = reactiveOmit(props as Record<string, unknown>, 'class')
const forwarded = useForwardProps(delegatedProps)
</script>

<template>
  <TabsTrigger
    v-bind="forwarded"
    :class="
      cn(
        buttonVariants({ variant: 'outline', size: 'sm' }),
        'data-[state=active]:bg-primary data-[state=active]:text-primary-foreground',
        props.class,
      )
    "
  >
    <slot />
  </TabsTrigger>
</template>
