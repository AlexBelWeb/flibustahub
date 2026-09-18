<script setup lang="ts">
import type { ToggleGroupRootEmits, ToggleGroupRootProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { ToggleGroupRoot, useForwardPropsEmits } from 'reka-ui'
import { cn, reactiveOmit } from '@/lib/utils'

const props = defineProps<ToggleGroupRootProps & { class?: HTMLAttributes['class'] }>()
const emits = defineEmits<ToggleGroupRootEmits>()
const delegatedProps = reactiveOmit(props as Record<string, unknown>, 'class')
const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <ToggleGroupRoot v-bind="forwarded" :class="cn('flex flex-wrap items-center gap-1', props.class)">
    <slot />
  </ToggleGroupRoot>
</template>
