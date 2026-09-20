<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ListState from '@/components/catalog/ListState.vue'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { errorMessage } from '@/i18n/errors'
import { parseBackendError } from '@/lib/backend-error'
import { formatCount, formatDate } from '@/lib/format'
import { usePersonalStore } from '@/stores/personal'
import { useToastStore } from '@/stores/toast'
import type { PersonalExportResult, PersonalImportReport } from '@/types/personal'
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
const personal = usePersonalStore()
const toast = useToastStore()

const exporting = ref(false)
const exportResult = ref<PersonalExportResult | null>(null)
const importPath = ref('')
const previewStatus = ref<'idle' | 'loading' | 'ready' | 'error'>('idle')
const previewError = ref('')
const preview = ref<PersonalImportReport | null>(null)
const confirmOpen = ref(false)
const applying = ref(false)
const applied = ref<PersonalImportReport | null>(null)

const lastExport = computed(() => {
  const at = personal.snapshot.lastExportAt
  return at ? formatDate(at, locale.value) : ''
})

onMounted(() => {
  if (personal.snapshotStatus === 'idle' || personal.snapshotStatus === 'error') {
    void personal.loadSnapshot()
  }
})

async function exportMarks() {
  try {
    const path = await window.go.handlers.App.SelectPersonalExportPath(
      t('settings.personal.exportTitle'),
    )
    if (!path) {
      return
    }
    exporting.value = true
    exportResult.value = await personal.exportTo(path)
  } catch (err) {
    const be = parseBackendError(err)
    toast.pushError(errorMessage(be.code, be.params))
  } finally {
    exporting.value = false
  }
}

async function pickImport() {
  try {
    const path = await window.go.handlers.App.SelectPersonalImportPath(
      t('settings.personal.importTitle'),
    )
    if (!path) {
      return
    }
    importPath.value = path
    applied.value = null
    previewStatus.value = 'loading'
    await loadPreview()
  } catch (err) {
    const be = parseBackendError(err)
    toast.pushError(errorMessage(be.code, be.params))
  }
}

async function loadPreview() {
  if (!importPath.value) {
    return
  }
  previewStatus.value = 'loading'
  previewError.value = ''
  preview.value = null
  try {
    preview.value = await personal.previewImport(importPath.value)
    const report = preview.value
    const total = report.applied + report.skipped + report.notFound + report.invalid
    previewStatus.value = total === 0 ? 'empty' : 'ready'
  } catch (err) {
    const be = parseBackendError(err)
    previewError.value = errorMessage(be.code, be.params)
    previewStatus.value = 'error'
  }
}

async function applyImport() {
  if (!importPath.value) {
    return
  }
  applying.value = true
  try {
    applied.value = await personal.applyImport(importPath.value)
    confirmOpen.value = false
  } catch (err) {
    const be = parseBackendError(err)
    toast.pushError(errorMessage(be.code, be.params))
  } finally {
    applying.value = false
  }
}

async function showNotFound() {
  const path = applied.value?.notFoundPath
  if (!path) {
    return
  }
  try {
    await window.go.handlers.App.ShowInFolder(path)
  } catch (err) {
    const be = parseBackendError(err)
    toast.pushError(errorMessage(be.code, be.params))
  }
}

function noteReason(reason: string) {
  switch (reason) {
    case 'stale':
      return t('settings.personal.reason.stale')
    case 'not_found':
      return t('settings.personal.reason.notFound')
    case 'invalid':
      return t('settings.personal.reason.invalid')
    case 'no_timestamp':
      return t('settings.personal.reason.noTimestamp')
    default:
      return reason
  }
}

function noteField(field: string) {
  switch (field) {
    case 'rating':
      return t('settings.personal.field.rating')
    case 'comment':
      return t('settings.personal.field.comment')
    case 'wantToRead':
      return t('settings.personal.field.wantToRead')
    default:
      return field
  }
}
</script>

<template>
  <section class="rounded-2xl border border-border bg-card/80 p-6 backdrop-panel">
    <h2 class="font-display text-xl font-medium">{{ t('settings.personal.title') }}</h2>
    <p class="mt-2 text-sm text-muted-foreground">{{ t('settings.personal.lead') }}</p>

    <div v-if="personal.snapshotStatus === 'loading'" class="mt-4 grid gap-2" aria-busy="true">
      <span class="sr-only">{{ t('common.loading') }}</span>
      <Skeleton class="h-6 w-1/2" />
      <Skeleton class="h-4 w-2/3" />
    </div>
    <div v-else-if="personal.snapshotStatus === 'error'" class="mt-4">
      <p class="mb-3">{{ personal.snapshotError || t('list.error') }}</p>
      <Button variant="outline" size="sm" @click="personal.loadSnapshot()">{{
        t('common.retry')
      }}</Button>
    </div>
    <dl v-else class="mt-4 grid gap-2 text-sm">
      <div>
        <dt class="text-muted-foreground">{{ t('settings.personal.unsyncedLabel') }}</dt>
        <dd class="tabular-nums">
          {{
            t('settings.personal.unsynced', personal.snapshot.unsyncedCount, {
              n: personal.snapshot.unsyncedCount,
            })
          }}
        </dd>
      </div>
      <div>
        <dt class="text-muted-foreground">{{ t('settings.personal.lastExportLabel') }}</dt>
        <dd>
          {{
            lastExport
              ? t('settings.personal.lastExport', { date: lastExport })
              : t('settings.personal.neverExported')
          }}
        </dd>
      </div>
    </dl>

    <div class="mt-6 flex flex-wrap gap-2">
      <Button :disabled="exporting" @click="exportMarks">{{
        t('settings.personal.export')
      }}</Button>
      <Button variant="outline" @click="pickImport">{{ t('settings.personal.import') }}</Button>
    </div>

    <p v-if="exportResult" class="mt-4 text-sm">
      {{
        t('settings.personal.exportDone', exportResult.count, {
          n: formatCount(exportResult.count, locale),
        })
      }}
      <span class="mt-1 block font-mono text-xs break-all text-muted-foreground">{{
        exportResult.path
      }}</span>
    </p>

    <div v-if="importPath" class="mt-8 grid gap-4">
      <h3 class="font-display text-lg font-medium">{{ t('settings.personal.previewTitle') }}</h3>
      <p class="font-mono text-xs break-all text-muted-foreground">{{ importPath }}</p>
      <ListState
        class="flex-none"
        :status="previewStatus"
        :empty-text="t('settings.personal.previewEmpty')"
        :error-text="previewError || t('list.error')"
        @retry="loadPreview"
      >
        <div v-if="preview" class="grid gap-2 text-sm">
          <p class="tabular-nums">
            {{ t('settings.personal.applied', preview.applied, { n: preview.applied }) }}
          </p>
          <p class="tabular-nums">
            {{ t('settings.personal.skipped', preview.skipped, { n: preview.skipped }) }}
          </p>
          <p class="tabular-nums">
            {{ t('settings.personal.notFound', preview.notFound, { n: preview.notFound }) }}
          </p>
          <p class="tabular-nums">
            {{ t('settings.personal.invalid', preview.invalid, { n: preview.invalid }) }}
          </p>
          <ul
            v-if="preview.notes?.length"
            class="mt-2 max-h-48 overflow-auto rounded-lg border border-border p-3 text-xs"
          >
            <li v-for="(note, index) in preview.notes" :key="index">
              {{
                t('settings.personal.noteLine', {
                  title: note.title || note.workKey || t('catalog.untitled'),
                  field: noteField(note.field || ''),
                  reason: noteReason(note.reason),
                })
              }}
            </li>
          </ul>
          <Button class="mt-2 w-fit" :disabled="!preview.applied" @click="confirmOpen = true">
            {{ t('settings.personal.apply') }}
          </Button>
        </div>
      </ListState>
    </div>

    <div v-if="applied" class="mt-8 grid gap-2 text-sm">
      <h3 class="font-display text-lg font-medium">{{ t('settings.personal.reportTitle') }}</h3>
      <p class="tabular-nums">
        {{ t('settings.personal.applied', applied.applied, { n: applied.applied }) }}
      </p>
      <p class="tabular-nums">
        {{ t('settings.personal.skipped', applied.skipped, { n: applied.skipped }) }}
      </p>
      <p class="tabular-nums">
        {{ t('settings.personal.notFound', applied.notFound, { n: applied.notFound }) }}
      </p>
      <p class="tabular-nums">
        {{ t('settings.personal.invalid', applied.invalid, { n: applied.invalid }) }}
      </p>
      <template v-if="applied.notFoundPath">
        <p class="font-mono text-xs break-all text-muted-foreground">{{ applied.notFoundPath }}</p>
        <Button variant="outline" size="sm" class="w-fit" @click="showNotFound">
          {{ t('book.showInFolder') }}
        </Button>
      </template>
    </div>
  </section>

  <AlertDialogRoot :open="confirmOpen" @update:open="confirmOpen = $event">
    <AlertDialogPortal>
      <AlertDialogOverlay class="fixed inset-0 z-[90] bg-black/50" />
      <AlertDialogContent
        class="fixed top-1/2 left-1/2 z-[91] w-[min(28rem,calc(100%-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-border bg-card p-6 shadow-lg"
      >
        <AlertDialogTitle class="font-display text-lg">{{
          t('settings.personal.confirmTitle')
        }}</AlertDialogTitle>
        <AlertDialogDescription class="mt-2 text-sm text-muted-foreground">
          {{ t('settings.personal.confirmBody') }}
        </AlertDialogDescription>
        <div class="mt-6 flex justify-end gap-2">
          <AlertDialogCancel as-child>
            <Button variant="outline">{{ t('common.cancel') }}</Button>
          </AlertDialogCancel>
          <AlertDialogAction as-child>
            <Button :disabled="applying" @click.prevent="applyImport">{{
              t('settings.personal.apply')
            }}</Button>
          </AlertDialogAction>
        </div>
      </AlertDialogContent>
    </AlertDialogPortal>
  </AlertDialogRoot>
</template>
