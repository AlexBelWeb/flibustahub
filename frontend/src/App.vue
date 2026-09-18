<script setup lang="ts">
import { onMounted } from 'vue'
import AppToaster from '@/components/AppToaster.vue'
import DatabaseUpdating from '@/components/DatabaseUpdating.vue'
import ImportModal from '@/components/import/ImportModal.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import StartupBlock from '@/components/StartupBlock.vue'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { TooltipProvider } from '@/components/ui/tooltip'
import { useAppStore } from '@/stores/app'
import { useImportStore } from '@/stores/import'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const app = useAppStore()
const imp = useImportStore()

onMounted(() => {
  imp.listen()
})
</script>

<template>
  <TooltipProvider>
    <div class="flex h-screen flex-col overflow-hidden bg-background text-foreground">
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
        <AppToaster />
      </template>
    </div>
  </TooltipProvider>
</template>
