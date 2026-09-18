<script setup lang="ts">
import type { ComboboxContentEmits, ComboboxContentProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { ComboboxContent, ComboboxViewport, useForwardPropsEmits } from 'reka-ui'
import { cn, reactiveOmit } from '@/lib/utils'

const props = withDefaults(
  defineProps<ComboboxContentProps & { class?: HTMLAttributes['class'] }>(),
  { position: 'inline' },
)
const emits = defineEmits<ComboboxContentEmits>()
const delegatedProps = reactiveOmit(props as Record<string, unknown>, 'class')
const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <ComboboxContent v-bind="forwarded" :class="cn('max-h-72 overflow-hidden', props.class)">
    <ComboboxViewport class="max-h-72 overflow-auto p-1">
      <slot />
    </ComboboxViewport>
  </ComboboxContent>
</template>
