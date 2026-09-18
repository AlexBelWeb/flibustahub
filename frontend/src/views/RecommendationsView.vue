<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
</script>

<template>
  <div class="mx-auto flex min-h-0 flex-1 flex-col justify-center px-6 py-10">
    <div class="rounded-2xl border border-border bg-card p-8">
      <h1 class="font-display text-3xl font-semibold">{{ t('recommendations.title') }}</h1>
      <h2 class="mt-6 text-xl">{{ t('recommendations.emptyTitle') }}</h2>
      <p class="mt-2 text-muted-foreground">{{ t('recommendations.emptyBody') }}</p>
    </div>
  </div>
</template>
