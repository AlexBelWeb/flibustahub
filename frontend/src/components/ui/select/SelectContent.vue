<script setup lang="ts">
import type { SelectContentEmits, SelectContentProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { SelectContent, SelectPortal, SelectViewport, useForwardPropsEmits } from 'reka-ui'
import { cn, reactiveOmit } from '@/lib/utils'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<SelectContentProps & { class?: HTMLAttributes['class'] }>(),
  {
    position: 'popper',
  },
)
const emits = defineEmits<SelectContentEmits>()
const delegatedProps = reactiveOmit(props as Record<string, unknown>, 'class')
const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <SelectPortal>
    <SelectContent
      v-bind="{ ...forwarded, ...$attrs }"
      :class="
        cn(
          'motion-pop elevate-raised relative z-50 max-h-96 min-w-32 overflow-hidden rounded-md bg-popover text-popover-foreground',
          props.position === 'popper' && 'min-w-(--reka-select-trigger-width)',
          props.class,
        )
      "
    >
      <SelectViewport :class="cn('p-1', props.position === 'popper' && 'w-full')">
        <slot />
      </SelectViewport>
    </SelectContent>
  </SelectPortal>
</template>
