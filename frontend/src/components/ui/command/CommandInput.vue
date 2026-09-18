<script setup lang="ts">
import type { ComboboxInputEmits, ComboboxInputProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { ComboboxInput, useForwardPropsEmits } from 'reka-ui'
import { Search } from '@lucide/vue'
import { cn, reactiveOmit } from '@/lib/utils'

defineOptions({ inheritAttrs: false })

const props = defineProps<ComboboxInputProps & { class?: HTMLAttributes['class'] }>()
const emits = defineEmits<ComboboxInputEmits>()
const delegatedProps = reactiveOmit(props as Record<string, unknown>, 'class')
const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <div class="flex items-center border-b border-border px-3" cmdk-input-wrapper>
    <Search class="mr-2 size-4 shrink-0 opacity-50" />
    <ComboboxInput
      v-bind="{ ...forwarded, ...$attrs }"
      auto-focus
      :class="
        cn(
          'flex h-10 w-full rounded-md bg-transparent py-3 text-sm outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50',
          props.class,
        )
      "
    />
  </div>
</template>
