<script setup lang="ts">
import type { ToggleGroupItemProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { ToggleGroupItem, useForwardProps } from 'reka-ui'
import { cn, reactiveOmit } from '@/lib/utils'
import { buttonVariants } from '@/components/ui/button'

const props = defineProps<ToggleGroupItemProps & { class?: HTMLAttributes['class'] }>()
const delegatedProps = reactiveOmit(props as Record<string, unknown>, 'class')
const forwarded = useForwardProps(delegatedProps)
</script>

<template>
  <ToggleGroupItem
    v-bind="forwarded"
    :class="
      cn(
        buttonVariants({ variant: 'outline', size: 'sm' }),
        'data-[state=on]:bg-primary data-[state=on]:text-primary-foreground',
        props.class,
      )
    "
  >
    <slot />
  </ToggleGroupItem>
</template>
