<script setup lang="ts">
import { RouterView } from 'vue-router'
import AppToaster from '@/components/AppToaster.vue'
import StartupBlock from '@/components/StartupBlock.vue'
import { useAppStore } from '@/stores/app'

const app = useAppStore()
</script>

<template>
  <div class="min-h-screen bg-background text-foreground">
    <div v-if="app.loading" class="min-h-screen" aria-busy="true" />
    <StartupBlock v-else-if="app.startupError" />
    <template v-else>
      <RouterView />
      <AppToaster />
    </template>
  </div>
</template>
