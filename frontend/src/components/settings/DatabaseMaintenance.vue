<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import IndeterminateProgress from '@/components/IndeterminateProgress.vue'
import { Button } from '@/components/ui/button'
import { errorMessage } from '@/i18n/errors'
import { formatBytes } from '@/lib/format'
import { parseBackendError } from '@/lib/backend-error'
import { useCoversStore } from '@/stores/covers'
import { useImportStore } from '@/stores/import'
import { useMaintenanceStore } from '@/stores/maintenance'
import { useToastStore } from '@/stores/toast'
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

const { t, locale } = useI18n()
const maint = useMaintenanceStore()
const imp = useImportStore()
const covers = useCoversStore()
const toast = useToastStore()
const { running, optimizeResult, backupResult } = storeToRefs(maint)
const confirmOptimize = ref(false)

const blockedReason = computed(() => {
  if (imp.running) {
    return t('errors.import_in_progress')
  }
  if (covers.progress.running) {
    return t('errors.cover_warmup_in_progress')
  }
  return ''
})

const disabled = computed(() => running.value || Boolean(blockedReason.value))

const optimizeLine = computed(() => {
  const result = optimizeResult.value
  if (!result) {
    return ''
  }
  if (result.bytesFreed <= 0) {
    return t('settings.maintenance.alreadyOptimized')
  }
  return t('settings.maintenance.freed', { size: formatBytes(result.bytesFreed, locale.value) })
})

onMounted(() => {
  void maint.refreshRunning()
})

async function runOptimize() {
  confirmOptimize.value = false
  try {
    await maint.optimize()
  } catch (err) {
    const be = parseBackendError(err)
    toast.pushError(errorMessage(be.code, be.params))
  }
}

async function runBackup() {
  try {
    await maint.backup()
  } catch (err) {
    const be = parseBackendError(err)
    toast.pushError(errorMessage(be.code, be.params))
  }
}
</script>

<template>
  <section class="mt-8 rounded-2xl border border-border bg-card/80 p-6 backdrop-panel">
    <h2 class="font-display text-xl font-medium">{{ t('settings.maintenance.title') }}</h2>
    <p class="mt-2 text-sm text-muted-foreground">{{ t('settings.maintenance.lead') }}</p>

    <p v-if="blockedReason" class="mt-4 text-sm">{{ blockedReason }}</p>

    <div v-if="running" class="mt-4 grid gap-2">
      <IndeterminateProgress :label="t('settings.maintenance.running')" />
      <p class="text-sm text-muted-foreground">{{ t('settings.maintenance.running') }}</p>
    </div>

    <div class="mt-4 flex flex-wrap gap-2">
      <Button :disabled="disabled" @click="confirmOptimize = true">
        {{ t('settings.maintenance.optimize') }}
      </Button>
      <Button variant="outline" :disabled="disabled" @click="runBackup">
        {{ t('settings.maintenance.backup') }}
      </Button>
    </div>

    <p v-if="optimizeLine" class="mt-4 text-sm tabular-nums">{{ optimizeLine }}</p>
    <p v-if="backupResult?.path" class="mt-2 font-mono text-sm break-all">
      {{ t('settings.maintenance.backupDone', { path: backupResult.path }) }}
    </p>
  </section>

  <AlertDialogRoot :open="confirmOptimize" @update:open="confirmOptimize = $event">
    <AlertDialogPortal>
      <AlertDialogOverlay class="fixed inset-0 z-[90] bg-scrim" />
      <AlertDialogContent
        class="fixed top-1/2 left-1/2 z-[90] w-[min(32rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-2xl dialog-surface border border-border p-6"
      >
        <AlertDialogTitle class="font-display text-lg">{{
          t('settings.maintenance.confirmTitle')
        }}</AlertDialogTitle>
        <AlertDialogDescription class="mt-2 text-sm text-muted-foreground">
          {{ t('settings.maintenance.confirmBody') }}
        </AlertDialogDescription>
        <div class="mt-6 flex flex-wrap justify-end gap-2">
          <AlertDialogCancel as-child>
            <Button variant="outline">{{ t('common.cancel') }}</Button>
          </AlertDialogCancel>
          <AlertDialogAction as-child>
            <Button :disabled="disabled" @click="runOptimize">{{
              t('settings.maintenance.optimize')
            }}</Button>
          </AlertDialogAction>
        </div>
      </AlertDialogContent>
    </AlertDialogPortal>
  </AlertDialogRoot>
</template>
