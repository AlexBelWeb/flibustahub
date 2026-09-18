<script setup lang="ts">
import type { ComboboxEmptyProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { ComboboxEmpty } from 'reka-ui'
import { cn, reactiveOmit } from '@/lib/utils'

const props = defineProps<ComboboxEmptyProps & { class?: HTMLAttributes['class'] }>()
const delegatedProps = reactiveOmit(props as Record<string, unknown>, 'class')
</script>

<template>
  <ComboboxEmpty
    v-bind="delegatedProps"
    :class="cn('py-6 text-center text-sm text-muted-foreground', props.class)"
  >
    <slot />
  </ComboboxEmpty>
</template>
