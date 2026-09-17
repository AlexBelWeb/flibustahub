<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { useToastStore } from '@/stores/toast'

const { t } = useI18n()
const toasts = useToastStore()
</script>

<template>
  <div
    class="pointer-events-none fixed right-4 bottom-4 z-50 flex w-80 max-w-[calc(100vw-2rem)] flex-col gap-2"
  >
    <div
      v-for="item in toasts.visible"
      :key="item.id"
      class="pointer-events-auto rounded-xl border border-border bg-card px-4 py-3 text-sm text-card-foreground shadow-lg"
      role="status"
    >
      <div class="flex items-start gap-3">
        <p class="min-w-0 flex-1">
          {{ item.message }}
          <span v-if="item.count > 1" class="text-muted-foreground tabular-nums">
            {{ t('toast.duplicateCount', { n: item.count }) }}
          </span>
        </p>
        <Button
          variant="ghost"
          size="sm"
          :aria-label="t('common.dismiss')"
          @click="toasts.dismiss(item.id)"
        >
          {{ t('common.dismiss') }}
        </Button>
      </div>
    </div>
  </div>
</template>
