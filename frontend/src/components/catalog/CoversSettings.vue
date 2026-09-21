<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { errorMessage } from '@/i18n/errors'
import { parseBackendError } from '@/lib/backend-error'
import { useCoversStore } from '@/stores/covers'
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
const covers = useCoversStore()
const { progress } = storeToRefs(covers)
const preview = ref<number | null>(null)
const previewStatus = ref<'idle' | 'loading' | 'ready' | 'error'>('idle')
const previewError = ref('')
const confirmClear = ref(false)
const busy = ref(false)

onMounted(() => {
  covers.listen()
  void loadPreview()
})

async function loadPreview() {
  previewStatus.value = 'loading'
  previewError.value = ''
  try {
    preview.value = await covers.preview()
    previewStatus.value = 'ready'
  } catch (err) {
    const be = parseBackendError(err)
    previewError.value = errorMessage(be.code, be.params)
    previewStatus.value = 'error'
  }
}

async function startWarmup() {
  busy.value = true
  try {
    await covers.start()
  } catch (err) {
    const be = parseBackendError(err)
    previewError.value = errorMessage(be.code, be.params)
    previewStatus.value = 'error'
  } finally {
    busy.value = false
  }
}

async function clearCache() {
  busy.value = true
  try {
    await covers.clear()
    confirmClear.value = false
    await loadPreview()
  } catch (err) {
    const be = parseBackendError(err)
    previewError.value = errorMessage(be.code, be.params)
    previewStatus.value = 'error'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <section class="mt-8 rounded-2xl border border-border bg-card/80 p-6 backdrop-panel">
    <h2 class="font-display text-xl font-medium">{{ t('settings.covers.title') }}</h2>
    <p class="mt-2 text-sm text-muted-foreground">{{ t('settings.covers.lead') }}</p>

    <div v-if="previewStatus === 'loading'" class="mt-4" aria-busy="true">
      <p>{{ t('common.loading') }}</p>
    </div>
    <div v-else-if="previewStatus === 'error'" class="mt-4">
      <p class="mb-3">{{ previewError || t('list.error') }}</p>
      <Button variant="outline" @click="loadPreview">{{ t('common.retry') }}</Button>
    </div>
    <div v-else class="mt-4 grid gap-4">
      <p>
        {{ t('settings.covers.warmupCount', preview ?? 0, { n: preview ?? 0 }) }}
      </p>
      <div class="flex flex-wrap gap-2">
        <Button
          v-if="!progress.running"
          :disabled="busy || (preview ?? 0) === 0"
          @click="startWarmup"
        >
          {{ t('settings.covers.warmupStart') }}
        </Button>
        <Button v-else variant="outline" @click="covers.stop()">
          {{ t('settings.covers.warmupStop') }}
        </Button>
        <Button variant="outline" :disabled="busy || progress.running" @click="confirmClear = true">
          {{ t('settings.covers.clear') }}
        </Button>
      </div>
      <div v-if="progress.running || progress.done > 0" class="grid gap-2">
        <p class="text-sm tabular-nums text-muted-foreground">
          {{ t('settings.covers.warmupProgress', { done: progress.done, total: progress.total }) }}
        </p>
        <div class="h-2 overflow-hidden rounded-full bg-muted">
          <div
            class="h-full bg-primary transition-[width] duration-200"
            :style="{
              width: progress.total
                ? `${Math.min(100, (100 * progress.done) / progress.total)}%`
                : '0%',
            }"
          />
        </div>
      </div>
    </div>
  </section>

  <AlertDialogRoot :open="confirmClear" @update:open="confirmClear = $event">
    <AlertDialogPortal>
      <AlertDialogOverlay class="fixed inset-0 z-[90] bg-scrim" />
      <AlertDialogContent
        class="fixed top-1/2 left-1/2 z-[90] w-[min(28rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-2xl dialog-surface border border-border p-6"
      >
        <AlertDialogTitle class="font-display text-lg">{{
          t('settings.covers.clearTitle')
        }}</AlertDialogTitle>
        <AlertDialogDescription class="mt-2 text-sm text-muted-foreground">
          {{ t('settings.covers.clearBody') }}
        </AlertDialogDescription>
        <div class="mt-6 flex justify-end gap-2">
          <AlertDialogCancel as-child>
            <Button variant="outline">{{ t('common.cancel') }}</Button>
          </AlertDialogCancel>
          <AlertDialogAction as-child>
            <Button :disabled="busy" @click="clearCache">{{ t('settings.covers.clear') }}</Button>
          </AlertDialogAction>
        </div>
      </AlertDialogContent>
    </AlertDialogPortal>
  </AlertDialogRoot>
</template>
