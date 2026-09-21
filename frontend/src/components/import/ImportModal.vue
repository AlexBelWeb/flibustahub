<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ImportReport from '@/components/import/ImportReport.vue'
import { Button } from '@/components/ui/button'
import { bytesPercent, formatCount, formatDuration } from '@/lib/format'
import { useImportStore } from '@/stores/import'
import {
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogOverlay,
  AlertDialogPortal,
  AlertDialogRoot,
  AlertDialogTitle,
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
  ProgressIndicator,
  ProgressRoot,
} from 'reka-ui'

const { t, locale } = useI18n()
const imp = useImportStore()

const loc = computed(() => locale.value)
const percent = computed(() =>
  bytesPercent(imp.progress?.bytesDone ?? 0, imp.progress?.bytesTotal ?? 0),
)
const determinate = computed(() => imp.visiblePhase === 'records')
const etaLabel = computed(() => {
  const p = imp.progress
  if (!p || p.bytesDone <= 0 || p.bytesTotal <= p.bytesDone || imp.progressStartedAt <= 0) {
    return ''
  }
  const elapsed = Date.now() - imp.progressStartedAt
  if (elapsed < 800) {
    return ''
  }
  const remainingMs = ((p.bytesTotal - p.bytesDone) * elapsed) / p.bytesDone
  return t('import.etaSeconds', { n: formatDuration(remainingMs, loc.value) })
})
const phaseTitle = computed(() => {
  if (imp.visiblePhase === 'fts') {
    return t('import.phase.fts')
  }
  if (imp.visiblePhase === 'warmup') {
    return t('import.phase.warmup')
  }
  return t('import.phase.records')
})

function onModalOpen(open: boolean) {
  if (!open) {
    imp.closeModal()
  }
}

function onEscape(event: Event) {
  if (imp.running) {
    event.preventDefault()
  }
}

function onInteractOutside(event: Event) {
  if (imp.running) {
    event.preventDefault()
  }
}

function onCloseDialog(open: boolean) {
  if (!open) {
    imp.stayInApp()
  }
}
</script>

<template>
  <DialogRoot :open="imp.modalOpen" @update:open="onModalOpen">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[60] bg-scrim" />
      <DialogContent
        class="fixed top-1/2 left-1/2 z-[60] flex max-h-[min(40rem,calc(100vh-2rem))] w-[min(40rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 flex-col rounded-2xl dialog-surface border border-border p-6"
        @escape-key-down="onEscape"
        @pointer-down-outside="onInteractOutside"
        @focus-outside="onInteractOutside"
      >
        <template v-if="imp.modalStage === 'progress'">
          <DialogTitle class="font-display text-2xl font-semibold">
            {{ t('import.runningTitle') }}
          </DialogTitle>
          <DialogDescription class="mt-2 text-muted-foreground">
            {{ phaseTitle }}
          </DialogDescription>

          <ol class="mt-6 grid gap-1 text-sm">
            <li
              :aria-current="imp.visiblePhase === 'records' ? 'step' : undefined"
              :class="imp.visiblePhase === 'records' ? 'text-foreground' : 'text-muted-foreground'"
            >
              {{ t('import.phase.records') }}
            </li>
            <li
              :aria-current="imp.visiblePhase === 'fts' ? 'step' : undefined"
              :class="imp.visiblePhase === 'fts' ? 'text-foreground' : 'text-muted-foreground'"
            >
              {{ t('import.phase.fts') }}
            </li>
            <li
              :aria-current="imp.visiblePhase === 'warmup' ? 'step' : undefined"
              :class="imp.visiblePhase === 'warmup' ? 'text-foreground' : 'text-muted-foreground'"
            >
              {{ t('import.phase.warmup') }}
            </li>
          </ol>

          <div class="mt-6 grid gap-4">
            <p class="tabular-nums" aria-live="polite">
              {{
                t('import.recordsProcessed', imp.progress?.recordsSeen ?? 0, {
                  n: formatCount(imp.progress?.recordsSeen ?? 0, loc),
                })
              }}
            </p>
            <p v-if="etaLabel && determinate" class="text-sm text-muted-foreground tabular-nums">
              {{ etaLabel }}
            </p>
            <ProgressRoot
              class="relative h-2 overflow-hidden rounded-full bg-muted"
              :model-value="determinate ? percent : null"
              :max="100"
            >
              <ProgressIndicator
                v-if="determinate"
                class="h-full w-full bg-primary transition-transform duration-200"
                :style="{ transform: `translateX(-${100 - percent}%)` }"
              />
              <div
                v-else
                class="import-pulse absolute inset-y-0 left-0 w-1/3 rounded-full bg-primary"
              />
            </ProgressRoot>
          </div>

          <div class="mt-8 flex flex-col gap-2">
            <Button variant="outline" :disabled="!imp.canCancel" @click="imp.requestCancel()">
              {{ t('import.cancel') }}
            </Button>
            <p v-if="!imp.canCancel && imp.running" class="text-sm text-muted-foreground">
              {{ t('import.cancelDisabled') }}
            </p>
          </div>
        </template>

        <template v-else>
          <DialogTitle class="font-display text-2xl font-semibold">
            {{ t('import.reportTitle') }}
          </DialogTitle>
          <DialogDescription class="sr-only">
            {{ t('import.reportTitle') }}
          </DialogDescription>
          <p v-if="imp.runError" class="mt-3">{{ imp.runErrorText() }}</p>
          <div class="mt-4 min-h-0 flex-1 overflow-auto pr-1">
            <ImportReport v-if="imp.result && imp.result.id" :report="imp.result" />
          </div>
          <div class="mt-6 flex justify-end">
            <Button @click="imp.closeModal()">{{ t('common.dismiss') }}</Button>
          </div>
        </template>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>

  <AlertDialogRoot :open="imp.confirmCancel" @update:open="imp.confirmCancel = $event">
    <AlertDialogPortal>
      <AlertDialogOverlay class="fixed inset-0 z-[80] bg-scrim" />
      <AlertDialogContent
        class="fixed top-1/2 left-1/2 z-[80] w-[min(28rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-2xl dialog-surface border border-border p-6"
      >
        <AlertDialogTitle class="font-display text-xl font-semibold">
          {{ t('import.cancelTitle') }}
        </AlertDialogTitle>
        <AlertDialogDescription class="mt-3 text-sm text-muted-foreground">
          {{ t('import.cancelBody') }}
        </AlertDialogDescription>
        <div class="mt-6 flex flex-wrap justify-end gap-2">
          <AlertDialogCancel as-child>
            <Button variant="outline">{{ t('common.stay') }}</Button>
          </AlertDialogCancel>
          <AlertDialogAction as-child>
            <Button @click="imp.confirmCancelImport()">{{ t('import.cancel') }}</Button>
          </AlertDialogAction>
        </div>
      </AlertDialogContent>
    </AlertDialogPortal>
  </AlertDialogRoot>

  <AlertDialogRoot :open="imp.confirmClose" @update:open="onCloseDialog">
    <AlertDialogPortal>
      <AlertDialogOverlay class="fixed inset-0 z-[80] bg-scrim" />
      <AlertDialogContent
        class="fixed top-1/2 left-1/2 z-[80] w-[min(32rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-2xl dialog-surface border border-border p-6"
      >
        <AlertDialogTitle class="font-display text-xl font-semibold">
          {{ imp.closeCommitted ? t('import.closeAfterTitle') : t('import.closeBeforeTitle') }}
        </AlertDialogTitle>
        <AlertDialogDescription class="mt-3 text-sm text-muted-foreground">
          {{ imp.closeCommitted ? t('import.closeAfterBody') : t('import.closeBeforeBody') }}
        </AlertDialogDescription>
        <div class="mt-6 flex flex-wrap justify-end gap-2">
          <AlertDialogCancel as-child>
            <Button variant="outline">{{ t('common.stay') }}</Button>
          </AlertDialogCancel>
          <AlertDialogAction as-child>
            <Button @click="imp.leaveApp()">{{ t('import.closeAnyway') }}</Button>
          </AlertDialogAction>
        </div>
      </AlertDialogContent>
    </AlertDialogPortal>
  </AlertDialogRoot>
</template>

<style scoped>
.import-pulse {
  animation: import-pulse 1s ease-in-out infinite;
}

@keyframes import-pulse {
  0% {
    transform: translateX(-20%);
    opacity: 0.55;
  }
  50% {
    transform: translateX(180%);
    opacity: 1;
  }
  100% {
    transform: translateX(-20%);
    opacity: 0.55;
  }
}
</style>
