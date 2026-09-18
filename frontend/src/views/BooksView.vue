<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import WorkBrowser from '@/components/catalog/WorkBrowser.vue'

const { t } = useI18n()
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col px-6 py-6">
    <h1 class="mb-4 font-display text-3xl font-semibold">{{ t('catalog.title') }}</h1>
    <WorkBrowser cache-key="books" show-random />
  </div>
</template>
