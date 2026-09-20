<script setup lang="ts">
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { useAppStore } from '@/stores/app'
import { useStorageStore } from '@/stores/storage'
import { useToastStore } from '@/stores/toast'
import { errorMessage } from '@/i18n/errors'
import { parseBackendError } from '@/lib/backend-error'

const { t } = useI18n()
const app = useAppStore()
const storage = useStorageStore()
const toast = useToastStore()

onMounted(() => {
  void storage.loadReaderPath()
})

async function chooseDownloads() {
  try {
    const path = await window.go.handlers.App.SelectDownloadsDir(t('settings.downloads.choose'))
    if (path && app.bootstrap) {
      app.bootstrap = {
        ...app.bootstrap,
        paths: { ...app.bootstrap.paths, downloadsDir: path },
      }
    }
  } catch (err) {
    const be = parseBackendError(err)
    toast.pushError(errorMessage(be.code, be.params))
  }
}

async function openDownloads() {
  try {
    await window.go.handlers.App.OpenDownloadsDir()
  } catch (err) {
    const be = parseBackendError(err)
    toast.pushError(errorMessage(be.code, be.params))
  }
}

async function chooseReader() {
  try {
    const path = await window.go.handlers.App.SelectReaderPath(t('settings.reader.choose'))
    if (path) {
      storage.readerPath = path
    }
  } catch (err) {
    const be = parseBackendError(err)
    toast.pushError(errorMessage(be.code, be.params))
  }
}

async function clearReader() {
  try {
    await window.go.handlers.App.ClearReaderPath()
    storage.readerPath = ''
  } catch (err) {
    const be = parseBackendError(err)
    toast.pushError(errorMessage(be.code, be.params))
  }
}
</script>

<template>
  <section class="rounded-2xl border border-border bg-card/80 p-6 backdrop-panel">
    <h2 class="font-display text-xl font-medium">{{ t('settings.downloads.title') }}</h2>
    <p class="mt-2 text-sm text-muted-foreground">{{ t('settings.downloads.lead') }}</p>
    <p class="mt-4 font-mono text-sm break-all">
      {{ app.bootstrap?.paths.downloadsDir || t('settings.downloads.empty') }}
    </p>
    <div class="mt-4 flex flex-wrap gap-2">
      <Button variant="outline" @click="chooseDownloads">{{
        t('settings.downloads.choose')
      }}</Button>
      <Button v-if="app.bootstrap?.paths.downloadsDir" variant="ghost" @click="openDownloads">{{
        t('settings.downloads.open')
      }}</Button>
    </div>
  </section>

  <section class="mt-8 rounded-2xl border border-border bg-card/80 p-6 backdrop-panel">
    <h2 class="font-display text-xl font-medium">{{ t('settings.reader.title') }}</h2>
    <p class="mt-2 text-sm text-muted-foreground">{{ t('settings.reader.lead') }}</p>
    <p class="mt-4 font-mono text-sm break-all">
      {{ storage.readerPath || t('settings.reader.osDefault') }}
    </p>
    <div class="mt-4 flex flex-wrap gap-2">
      <Button variant="outline" @click="chooseReader">{{ t('settings.reader.choose') }}</Button>
      <Button v-if="storage.readerPath" variant="ghost" @click="clearReader">{{
        t('settings.reader.clear')
      }}</Button>
    </div>
  </section>
</template>
