<script setup lang="ts">
import { Bookmark, BookmarkCheck } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

const props = defineProps<{
  modelValue: boolean
  compact?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const { t } = useI18n()

function onClick(event: MouseEvent) {
  event.stopPropagation()
  emit('update:modelValue', !props.modelValue)
}
</script>

<template>
  <Tooltip>
    <TooltipTrigger as-child>
      <Button
        type="button"
        :variant="modelValue ? 'default' : 'outline'"
        :size="compact ? 'icon' : 'sm'"
        :aria-pressed="modelValue"
        :aria-label="t('personal.want')"
        @click="onClick"
      >
        <BookmarkCheck v-if="modelValue" class="size-4 fill-current" />
        <Bookmark v-else class="size-4" />
        <span v-if="!compact">{{ t('personal.want') }}</span>
      </Button>
    </TooltipTrigger>
    <TooltipContent>
      {{ modelValue ? t('personal.wantHintOn') : t('personal.wantHintOff') }}
    </TooltipContent>
  </Tooltip>
</template>
