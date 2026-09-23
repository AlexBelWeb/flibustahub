<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
</script>

<template>
  <div
    class="scroll-stable mx-auto flex min-h-0 w-full max-w-6xl flex-1 flex-col justify-center overflow-auto px-6 py-10"
  >
    <div class="elevate rounded-2xl bg-card p-8">
      <h1 class="type-page">{{ t('recommendations.title') }}</h1>
      <h2 class="mt-6 text-xl">{{ t('recommendations.emptyTitle') }}</h2>
      <p class="mt-2 text-muted-foreground">{{ t('recommendations.emptyBody') }}</p>
    </div>
  </div>
</template>
