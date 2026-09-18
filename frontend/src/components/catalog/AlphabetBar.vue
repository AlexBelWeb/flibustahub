<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { letterLabel } from '@/lib/letter'

const props = defineProps<{
  letters: string[]
  active: string
}>()

const emit = defineEmits<{
  select: [letter: string]
}>()

const { t } = useI18n()

function label(key: string) {
  return letterLabel(key, t('alphabet.yo'), t('alphabet.other'))
}

const items = computed(() => props.letters)
</script>

<template>
  <ToggleGroup
    type="single"
    class="justify-start"
    :model-value="active || undefined"
    @update:model-value="(value) => emit('select', value ? String(value) : '')"
  >
    <ToggleGroupItem v-for="letter in items" :key="letter" :value="letter" class="min-w-8 px-2">
      {{ label(letter) }}
    </ToggleGroupItem>
  </ToggleGroup>
</template>
