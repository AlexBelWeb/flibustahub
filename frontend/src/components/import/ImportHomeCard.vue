<script setup lang="ts">
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ImportReport from '@/components/import/ImportReport.vue'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { errorMessage } from '@/i18n/errors'
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
} from 'reka-ui'

const { t } = useI18n()
const imp = useImportStore()

onMounted(() => {
  void imp.loadCard()
})

function chooseFolder() {
  void imp.chooseFolder(t('import.chooseFolderTitle'))
}

function chooseDump() {
  void imp.chooseDump(t('import.chooseDumpTitle'))
}
</script>

<template>
  <section class="rounded-2xl border border-border bg-card/80 p-6 backdrop-panel">
    <h2 class="font-display text-xl font-medium">{{ t('import.title') }}</h2>

    <div v-if="imp.cardLoading" class="mt-4 grid min-h-56 gap-3" aria-busy="true">
      <Skeleton class="h-6 w-1/3" />
      <Skeleton class="h-4 w-2/3" />
      <Skeleton class="h-10 w-48" />
      <Skeleton class="h-24 rounded-xl" />
    </div>

    <div v-else-if="!imp.hasFolder" class="mt-4 grid min-h-56 content-start gap-3">
      <h3 class="text-lg">{{ t('import.emptyTitle') }}</h3>
      <p class="text-muted-foreground">{{ t('import.emptyBody') }}</p>
      <Button class="mt-2 w-fit" @click="chooseFolder">{{ t('import.chooseFolder') }}</Button>
    </div>

    <div v-else-if="imp.previewError" class="mt-4 grid min-h-56 content-start gap-3">
      <p>{{ errorMessage(imp.previewError.code, imp.previewError.params) }}</p>
      <div class="flex flex-wrap gap-2">
        <Button @click="imp.loadCard()">{{ t('common.retry') }}</Button>
        <Button variant="outline" @click="chooseFolder">{{ t('import.changeFolder') }}</Button>
      </div>
    </div>

    <div v-else-if="imp.preview" class="mt-4 grid min-h-56 content-start gap-4">
      <dl class="grid gap-2 text-sm">
        <div class="grid gap-1">
          <dt class="text-muted-foreground">{{ t('import.folder') }}</dt>
          <dd class="font-mono break-all">{{ imp.preview.libraryRoot }}</dd>
        </div>
        <div class="grid gap-1">
          <dt class="text-muted-foreground">{{ t('import.zipCount') }}</dt>
          <dd class="tabular-nums">{{ imp.preview.zipCount }}</dd>
        </div>
        <div class="grid gap-1">
          <dt class="text-muted-foreground">{{ t('import.dumpFile') }}</dt>
          <dd class="font-mono break-all">{{ imp.preview.inpxFileName }}</dd>
        </div>
        <div class="grid gap-1">
          <dt class="text-muted-foreground">{{ t('import.dumpVersion') }}</dt>
          <dd class="tabular-nums">{{ imp.preview.fileVersion }}</dd>
        </div>
      </dl>
      <div v-if="imp.preview.inpxFiles?.length > 1" class="grid gap-2">
        <button
          v-for="file in imp.preview.inpxFiles"
          :key="file.path"
          type="button"
          class="rounded-lg border border-border px-3 py-2 text-left font-mono text-sm hover:bg-accent"
          :class="imp.preview.inpxPath === file.path ? 'border-primary' : ''"
          @click="imp.pickDump(file.path)"
        >
          {{ file.name }}
        </button>
      </div>
      <div class="flex flex-wrap gap-2">
        <Button :disabled="imp.running" @click="imp.requestStart()">{{ t('import.start') }}</Button>
        <Button variant="outline" :disabled="imp.running" @click="chooseFolder">
          {{ t('import.changeFolder') }}
        </Button>
        <Button variant="outline" :disabled="imp.running" @click="chooseDump">
          {{ t('onboarding.pickManual') }}
        </Button>
      </div>

      <div v-if="imp.lastReport" class="border-t border-border pt-4">
        <h3 class="mb-3 font-medium">{{ t('import.lastTitle') }}</h3>
        <ImportReport :report="imp.lastReport" compact />
      </div>
      <p v-else class="text-sm text-muted-foreground">{{ t('import.lastNone') }}</p>
    </div>
  </section>

  <AlertDialogRoot :open="imp.confirmReimport" @update:open="imp.confirmReimport = $event">
    <AlertDialogPortal>
      <AlertDialogOverlay class="fixed inset-0 z-[70] bg-black/50" />
      <AlertDialogContent
        class="fixed top-1/2 left-1/2 z-[70] w-[min(32rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-border bg-card p-6 shadow-lg"
      >
        <AlertDialogTitle class="font-display text-xl font-semibold">
          {{ t('import.confirmTitle') }}
        </AlertDialogTitle>
        <AlertDialogDescription class="mt-3 grid gap-2 text-sm text-muted-foreground">
          <span v-if="imp.preview?.catalogVersion">
            {{ t('import.confirmCurrentVersion', { version: imp.preview.catalogVersion }) }}
          </span>
          <span v-if="imp.preview?.fileVersion">
            {{ t('import.confirmFileVersion', { version: imp.preview.fileVersion }) }}
          </span>
          <span>{{ t('import.confirmBody') }}</span>
          <span v-if="imp.preview?.sameVersion" class="text-foreground">
            {{ t('import.confirmSameVersion') }}
          </span>
        </AlertDialogDescription>
        <div class="mt-6 flex flex-wrap justify-end gap-2">
          <AlertDialogCancel as-child>
            <Button variant="outline">{{ t('common.dismiss') }}</Button>
          </AlertDialogCancel>
          <AlertDialogAction as-child>
            <Button @click="imp.runImport()">{{ t('common.confirm') }}</Button>
          </AlertDialogAction>
        </div>
      </AlertDialogContent>
    </AlertDialogPortal>
  </AlertDialogRoot>
</template>
