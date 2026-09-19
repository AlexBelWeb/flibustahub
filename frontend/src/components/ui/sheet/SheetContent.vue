<script setup lang="ts">
import type { DialogContentEmits, DialogContentProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import {
  DialogClose,
  DialogContent,
  DialogOverlay,
  DialogPortal,
  useForwardPropsEmits,
} from 'reka-ui'
import { useI18n } from 'vue-i18n'
import { X } from '@lucide/vue'
import { cn, reactiveOmit } from '@/lib/utils'
import { sheetVariants, type SheetVariants } from '.'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<
    DialogContentProps & {
      class?: HTMLAttributes['class']
      side?: SheetVariants['side']
      hideClose?: boolean
    }
  >(),
  { side: 'right', hideClose: false },
)
const emits = defineEmits<DialogContentEmits>()
const { t } = useI18n()
const delegatedProps = reactiveOmit(props as Record<string, unknown>, 'class', 'side', 'hideClose')
const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <DialogPortal>
    <DialogOverlay class="fixed inset-0 z-50 bg-black/80" />
    <DialogContent
      v-bind="{ ...forwarded, ...$attrs }"
      :class="cn(sheetVariants({ side: props.side }), props.class)"
    >
      <slot />
      <DialogClose
        v-if="!props.hideClose"
        class="absolute top-4 right-4 rounded-sm opacity-70 hover:opacity-100"
        :aria-label="t('common.dismiss')"
      >
        <X class="size-4" />
      </DialogClose>
    </DialogContent>
  </DialogPortal>
</template>
