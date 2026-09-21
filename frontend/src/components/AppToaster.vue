<script setup lang="ts">
import { CircleAlert, CircleCheck } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import {
  Toast,
  ToastClose,
  ToastDescription,
  ToastProvider,
  ToastViewport,
} from '@/components/ui/toast'
import { useToastStore } from '@/stores/toast'

const { t } = useI18n()
const toasts = useToastStore()
</script>

<template>
  <ToastProvider :label="t('toast.region')">
    <Toast
      v-for="item in toasts.visible"
      :key="item.id"
      :open="true"
      :duration="Infinity"
      :type="item.kind === 'error' ? 'foreground' : 'background'"
      @update:open="(open) => !open && toasts.dismiss(item.id)"
    >
      <div class="flex items-start gap-3">
        <CircleAlert
          v-if="item.kind === 'error'"
          class="mt-0.5 size-4 shrink-0 text-destructive"
          aria-hidden="true"
        />
        <CircleCheck
          v-else-if="item.kind === 'success'"
          class="mt-0.5 size-4 shrink-0 text-success"
          aria-hidden="true"
        />
        <ToastDescription
          :class="
            item.kind === 'error'
              ? 'text-destructive'
              : item.kind === 'success'
                ? 'text-success'
                : ''
          "
        >
          {{ item.message }}
          <span v-if="item.count > 1" class="text-muted-foreground tabular-nums">
            {{ t('toast.duplicateCount', { n: item.count }) }}
          </span>
        </ToastDescription>
        <Button
          v-if="item.onAction && item.actionLabel"
          variant="outline"
          size="sm"
          @click="toasts.runAction(item.id)"
        >
          {{ item.actionLabel }}
        </Button>
        <ToastClose as-child>
          <Button variant="ghost" size="sm" :aria-label="t('common.dismiss')">
            {{ t('common.dismiss') }}
          </Button>
        </ToastClose>
      </div>
    </Toast>
    <ToastViewport :label="t('toast.region')" />
  </ToastProvider>
</template>
