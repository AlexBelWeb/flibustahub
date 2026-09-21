<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { useMaintenanceStore } from '@/stores/maintenance'
import {
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogOverlay,
  AlertDialogPortal,
  AlertDialogRoot,
  AlertDialogTitle,
} from 'reka-ui'

const { t } = useI18n()
const maint = useMaintenanceStore()

function onCloseDialog(open: boolean) {
  if (!open) {
    maint.stayInApp()
  }
}
</script>

<template>
  <AlertDialogRoot :open="maint.confirmClose" @update:open="onCloseDialog">
    <AlertDialogPortal>
      <AlertDialogOverlay class="fixed inset-0 z-[80] bg-scrim" />
      <AlertDialogContent
        class="fixed top-1/2 left-1/2 z-[80] w-[min(32rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-2xl dialog-surface border border-border p-6"
      >
        <AlertDialogTitle class="font-display text-xl font-semibold">
          {{ t('settings.maintenance.closeTitle') }}
        </AlertDialogTitle>
        <AlertDialogDescription class="mt-3 text-sm text-muted-foreground">
          {{ t('settings.maintenance.closeBody') }}
        </AlertDialogDescription>
        <div class="mt-6 flex flex-wrap justify-end gap-2">
          <AlertDialogCancel as-child>
            <Button variant="outline">{{ t('common.stay') }}</Button>
          </AlertDialogCancel>
          <AlertDialogAction as-child>
            <Button @click="maint.leaveApp()">{{ t('settings.maintenance.closeAnyway') }}</Button>
          </AlertDialogAction>
        </div>
      </AlertDialogContent>
    </AlertDialogPortal>
  </AlertDialogRoot>
</template>
