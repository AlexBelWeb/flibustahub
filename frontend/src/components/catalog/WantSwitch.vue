<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Switch } from '@/components/ui/switch'

defineProps<{
  modelValue: boolean
  compact?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const { t } = useI18n()
</script>

<template>
  <label class="inline-flex items-center gap-2 text-sm" @click.stop>
    <Switch
      :model-value="modelValue"
      :aria-label="t('personal.want')"
      @update:model-value="(value) => emit('update:modelValue', Boolean(value))"
    />
    <span v-if="!compact">{{ t('personal.want') }}</span>
  </label>
</template>
