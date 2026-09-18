<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { errorMessage } from '@/i18n/errors'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const app = useAppStore()
const busy = ref(false)

const detail = computed(() => {
  const err = app.startupError
  if (!err) {
    return ''
  }
  return errorMessage(err.code, err.params ?? {})
})

async function retry() {
  busy.value = true
  try {
    await app.retryStartup()
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="mx-auto flex min-h-screen max-w-xl flex-col justify-center gap-6 px-6">
    <div class="rounded-2xl border border-border bg-card p-8">
      <h1 class="font-display text-3xl font-semibold">{{ t('startup.title') }}</h1>
      <p class="mt-3 text-muted-foreground">{{ t('startup.lead') }}</p>
      <p class="mt-4 text-base">{{ detail }}</p>
      <div class="mt-8 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
        <Button :disabled="busy || app.catalogOpening" @click="retry">{{
          t('common.retry')
        }}</Button>
        <Button variant="outline" :disabled="busy" @click="app.openLogsDir()">
          {{ t('startup.openLogs') }}
        </Button>
        <Button variant="outline" :disabled="busy" @click="app.openDataDir()">
          {{ t('startup.openData') }}
        </Button>
      </div>
    </div>
  </div>
</template>
