<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import InterfaceControls from '@/components/InterfaceControls.vue'
import { Button } from '@/components/ui/button'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const app = useAppStore()
</script>

<template>
  <div class="mx-auto flex min-h-full max-w-3xl flex-col gap-8 px-6 py-10">
    <header class="flex items-center justify-between gap-4">
      <div>
        <p class="text-sm text-muted-foreground">{{ t('settings.title') }}</p>
        <h1 class="font-display text-3xl font-semibold">{{ t('settings.interface.title') }}</h1>
      </div>
      <Button as-child variant="outline">
        <RouterLink to="/">{{ t('nav.home') }}</RouterLink>
      </Button>
    </header>

    <div v-if="app.loading" class="h-48 animate-pulse rounded-2xl bg-muted" />
    <div v-else-if="app.loadError" class="rounded-2xl border border-border bg-card p-6">
      <p class="mb-4">{{ t('errors.internal') }}</p>
      <Button @click="app.load()">{{ t('common.retry') }}</Button>
    </div>
    <section v-else class="rounded-2xl border border-border bg-card/80 p-6 backdrop-panel">
      <InterfaceControls />
    </section>
  </div>
</template>
