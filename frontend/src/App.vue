<script setup lang="ts">
import { onMounted } from 'vue'
import { RouterView } from 'vue-router'
import AppToaster from '@/components/AppToaster.vue'
import ImportModal from '@/components/import/ImportModal.vue'
import StartupBlock from '@/components/StartupBlock.vue'
import { useAppStore } from '@/stores/app'
import { useImportStore } from '@/stores/import'

const app = useAppStore()
const imp = useImportStore()

onMounted(() => {
  imp.listen()
})
</script>

<template>
  <div class="min-h-screen bg-background text-foreground">
    <div v-if="app.loading || app.databaseUpdating" class="min-h-screen" aria-busy="true" />
    <StartupBlock v-else-if="app.startupError" />
    <div
      v-else-if="app.catalogOpening || !app.catalogReady"
      class="min-h-screen"
      aria-busy="true"
    />
    <template v-else>
      <RouterView />
      <ImportModal />
      <AppToaster />
    </template>
  </div>
</template>
