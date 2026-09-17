<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import InterfaceControls from '@/components/InterfaceControls.vue'
import { Button } from '@/components/ui/button'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const app = useAppStore()

const effectsLabel = computed(() =>
  app.reducedEffects ? t('status.effectsReduced') : t('status.effectsFull'),
)
</script>

<template>
  <div class="mx-auto flex min-h-full max-w-3xl flex-col gap-8 px-6 py-10">
    <header class="grid gap-2">
      <p class="font-display text-sm tracking-wide text-muted-foreground uppercase">
        {{ t('app.name') }}
      </p>
      <p class="text-muted-foreground">{{ t('app.tagline') }}</p>
      <h1 class="font-display text-4xl font-semibold tracking-tight">{{ t('home.title') }}</h1>
      <p class="max-w-2xl text-base text-muted-foreground">{{ t('home.lead') }}</p>
    </header>

    <div v-if="app.loading" class="grid gap-3" aria-busy="true">
      <div class="h-24 animate-pulse rounded-2xl bg-muted" />
      <div class="h-40 animate-pulse rounded-2xl bg-muted" />
    </div>

    <div v-else-if="app.loadError" class="rounded-2xl border border-border bg-card p-6">
      <p class="mb-4">{{ t('errors.internal') }}</p>
      <Button @click="app.load()">{{ t('common.retry') }}</Button>
    </div>

    <template v-else>
      <section class="rounded-2xl border border-border bg-card/80 p-6 backdrop-panel">
        <InterfaceControls />
      </section>

      <section class="rounded-2xl border border-dashed border-border p-6">
        <h2 class="font-display text-xl font-medium">{{ t('home.emptyTitle') }}</h2>
        <p class="mt-2 text-muted-foreground">{{ t('home.emptyBody') }}</p>
        <Button as-child class="mt-4" variant="outline">
          <RouterLink to="/settings/interface">{{ t('nav.settings') }}</RouterLink>
        </Button>
      </section>

      <dl class="grid gap-2 font-mono text-xs text-muted-foreground tabular-nums">
        <div class="flex gap-2">
          <dt>{{ t('status.version') }}</dt>
          <dd>{{ app.bootstrap?.version }}</dd>
        </div>
        <div class="flex gap-2">
          <dt>{{ t('status.platform') }}</dt>
          <dd>{{ app.bootstrap?.capabilities.os }}</dd>
        </div>
        <div class="flex gap-2">
          <dt>{{ t('status.effects') }}</dt>
          <dd>{{ effectsLabel }}</dd>
        </div>
      </dl>
    </template>
  </div>
</template>
