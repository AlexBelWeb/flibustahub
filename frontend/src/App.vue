<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import AppToaster from '@/components/AppToaster.vue'
import DatabaseUpdating from '@/components/DatabaseUpdating.vue'
import ImportModal from '@/components/import/ImportModal.vue'
import MaintenanceCloseDialog from '@/components/settings/MaintenanceCloseDialog.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import StartupBlock from '@/components/StartupBlock.vue'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { TooltipProvider } from '@/components/ui/tooltip'
import { useAppStore } from '@/stores/app'
import { useCoversStore } from '@/stores/covers'
import { useImportStore } from '@/stores/import'
import { useMaintenanceStore } from '@/stores/maintenance'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const route = useRoute()
const palettePreview = computed(
  () => route.name === 'palettePreview' || window.location.hash.startsWith('#/palette-preview'),
)
const app = useAppStore()
const imp = useImportStore()
const covers = useCoversStore()
const maint = useMaintenanceStore()

function onWindowAway() {
  void window.go.handlers.App.WindowAway()
}

function onWindowBack() {
  void window.go.handlers.App.WindowBack()
}

function onVisibility() {
  if (document.visibilityState === 'hidden') {
    onWindowAway()
    return
  }
  onWindowBack()
}

onMounted(() => {
  imp.listen()
  covers.listen()
  maint.listen()
  window.addEventListener('blur', onWindowAway)
  window.addEventListener('focus', onWindowBack)
  document.addEventListener('visibilitychange', onVisibility)
})

onUnmounted(() => {
  window.removeEventListener('blur', onWindowAway)
  window.removeEventListener('focus', onWindowBack)
  document.removeEventListener('visibilitychange', onVisibility)
})
</script>

<template>
  <TooltipProvider>
    <div v-if="palettePreview" class="h-screen overflow-hidden">
      <RouterView />
    </div>
    <div v-else class="flex h-screen flex-col overflow-hidden bg-background text-foreground">
      <div v-if="app.loading" class="h-full" aria-busy="true" />
      <DatabaseUpdating v-else-if="app.databaseUpdating && !app.startupError" />
      <StartupBlock v-else-if="app.startupError" />
      <div
        v-else-if="app.loadError"
        class="mx-auto flex h-full max-w-xl flex-col justify-center px-6"
      >
        <Card class="p-8">
          <p class="mb-4">{{ t('errors.internal') }}</p>
          <Button @click="app.load()">{{ t('common.retry') }}</Button>
        </Card>
      </div>
      <div v-else-if="app.catalogOpening || !app.catalogReady" class="h-full" aria-busy="true" />
      <template v-else>
        <div class="flex min-h-0 flex-1 flex-col overflow-hidden">
          <AppLayout />
        </div>
        <ImportModal />
        <MaintenanceCloseDialog />
        <AppToaster />
      </template>
    </div>
  </TooltipProvider>
</template>
