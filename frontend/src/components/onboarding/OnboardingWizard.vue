<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { errorMessage } from '@/i18n/errors'
import { useImportStore } from '@/stores/import'
import { useUiStore } from '@/stores/ui'
import { DialogContent, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'

const { t } = useI18n()
const ui = useUiStore()
const imp = useImportStore()
const step = ref(0)

const open = computed({
  get: () => ui.onboardingOpen,
  set: (value: boolean) => {
    if (!value) {
      ui.skipOnboarding()
    } else {
      ui.onboardingOpen = true
    }
  },
})

watch(
  () => ui.onboardingOpen,
  (value) => {
    if (value) {
      step.value = imp.hasFolder ? (imp.preview?.inpxFileName ? 2 : 1) : 0
      if (imp.hasFolder) {
        void imp.loadCard()
      }
    }
  },
)

const zipLabel = computed(() => {
  const n = imp.preview?.zipCount ?? 0
  if (n === 0) {
    return t('onboarding.zipNone')
  }
  return t('onboarding.zipCount', n, { n })
})

function chooseFolder() {
  void imp.chooseFolder(t('import.chooseFolderTitle')).then(() => {
    if (imp.preview && !imp.previewError) {
      step.value = 1
    }
  })
}

function pickDump(path: string) {
  void imp.pickDump(path).then(() => {
    if (!imp.previewError) {
      step.value = 2
    }
  })
}

function pickManual() {
  void imp.chooseDump(t('import.chooseDumpTitle')).then(() => {
    if (imp.preview?.inpxFileName && !imp.previewError) {
      step.value = 2
    }
  })
}

function startImport() {
  ui.skipOnboarding()
  imp.requestStart()
}
</script>

<template>
  <DialogRoot :open="open" @update:open="open = $event">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[75] bg-scrim" />
      <DialogContent
        class="fixed top-1/2 left-1/2 z-[75] w-[min(36rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-2xl dialog-surface border border-border p-6"
      >
        <DialogTitle class="font-display text-2xl font-semibold">{{
          t('onboarding.title')
        }}</DialogTitle>
        <ol class="mt-4 flex gap-3 text-sm text-muted-foreground">
          <li :class="step === 0 ? 'text-foreground' : ''">{{ t('onboarding.stepFolder') }}</li>
          <li :class="step === 1 ? 'text-foreground' : ''">{{ t('onboarding.stepDump') }}</li>
          <li :class="step === 2 ? 'text-foreground' : ''">{{ t('onboarding.stepImport') }}</li>
        </ol>

        <div v-if="step === 0" class="mt-6 grid gap-4">
          <p>{{ t('onboarding.folderLead') }}</p>
          <p v-if="imp.previewError">
            {{ errorMessage(imp.previewError.code, imp.previewError.params) }}
          </p>
          <p v-else-if="imp.preview">{{ zipLabel }}</p>
          <div class="flex flex-wrap gap-2">
            <Button @click="chooseFolder">{{ t('onboarding.chooseFolder') }}</Button>
            <Button v-if="imp.preview && !imp.previewError" variant="outline" @click="step = 1">
              {{ t('onboarding.next') }}
            </Button>
          </div>
        </div>

        <div v-else-if="step === 1" class="mt-6 grid gap-4">
          <p>{{ t('onboarding.dumpLead') }}</p>
          <p v-if="imp.previewError">
            {{ errorMessage(imp.previewError.code, imp.previewError.params) }}
          </p>
          <p v-else-if="!imp.preview?.inpxFiles?.length">{{ t('onboarding.dumpNone') }}</p>
          <div v-else class="grid gap-2">
            <button
              v-for="file in imp.preview.inpxFiles"
              :key="file.path"
              type="button"
              class="rounded-lg border border-border px-3 py-2 text-left font-mono text-sm hover:bg-accent"
              :class="imp.preview.inpxPath === file.path ? 'border-primary' : ''"
              @click="pickDump(file.path)"
            >
              {{ file.name }}
            </button>
          </div>
          <div class="flex flex-wrap gap-2">
            <Button variant="outline" @click="pickManual">{{ t('onboarding.pickManual') }}</Button>
            <Button v-if="imp.preview?.inpxFileName" @click="step = 2">{{
              t('onboarding.next')
            }}</Button>
          </div>
        </div>

        <div v-else class="mt-6 grid gap-4">
          <p>{{ t('onboarding.importLead') }}</p>
          <p v-if="imp.preview?.inpxFileName" class="font-mono text-sm">
            {{ imp.preview.inpxFileName }}
          </p>
          <Button :disabled="imp.running" @click="startImport">{{ t('import.start') }}</Button>
        </div>

        <div class="mt-6 flex justify-end gap-2">
          <Button variant="ghost" @click="ui.skipOnboarding()">{{ t('onboarding.skip') }}</Button>
          <Button variant="outline" @click="ui.skipOnboarding()">{{
            t('onboarding.close')
          }}</Button>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
