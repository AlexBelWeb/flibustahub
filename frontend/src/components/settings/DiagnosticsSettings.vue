<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import IssueReportForm from '@/components/settings/IssueReportForm.vue'
import IndeterminateProgress from '@/components/IndeterminateProgress.vue'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { errorMessage } from '@/i18n/errors'
import { formatCount, formatDate, formatDumpVersionDate, formatRelative } from '@/lib/format'
import { copyText } from '@/lib/wails-runtime'
import { parseBackendError } from '@/lib/backend-error'
import { useAppStore } from '@/stores/app'
import { useDiagnosticsStore } from '@/stores/diagnostics'
import { useToastStore } from '@/stores/toast'

const { t, locale } = useI18n()
const diag = useDiagnosticsStore()
const app = useAppStore()
const toast = useToastStore()
const { snapshot, status, archiveBusy, archivePath } = storeToRefs(diag)
const copiedKey = ref('')

const webView = computed(() => {
  const raw = snapshot.value?.webView?.trim() ?? ''
  if (!raw || raw === 'unknown') {
    return t('settings.diagnostics.webViewUnknown')
  }
  return raw
})

const importedLabel = computed(() => {
  const iso = snapshot.value?.importedAt ?? ''
  if (iso) {
    return formatRelative(iso, locale.value)
  }
  return formatDumpVersionDate(snapshot.value?.inpxVersion, locale.value)
})

const importedExact = computed(() => {
  const iso = snapshot.value?.importedAt ?? ''
  if (iso) {
    return formatDate(iso, locale.value)
  }
  return ''
})

onMounted(() => {
  void diag.load()
})

async function copy(value: string, key: string) {
  const ok = await copyText(value)
  if (!ok) {
    toast.pushError(t('settings.issue.copyFailed'))
    return
  }
  copiedKey.value = key
  window.setTimeout(() => {
    if (copiedKey.value === key) {
      copiedKey.value = ''
    }
  }, 2000)
}

async function saveArchive() {
  try {
    await diag.saveArchive(t('settings.diagnostics.archiveTitle'))
  } catch (err) {
    const be = parseBackendError(err)
    toast.pushError(errorMessage(be.code, be.params))
  }
}

async function showArchive() {
  if (!archivePath.value) {
    return
  }
  try {
    await window.go.handlers.App.ShowInFolder(archivePath.value)
  } catch (err) {
    const be = parseBackendError(err)
    toast.pushError(errorMessage(be.code, be.params))
  }
}

const rows = computed(() => {
  const snap = snapshot.value
  if (!snap) {
    return []
  }
  return [
    { key: 'db', label: t('settings.diagnostics.dbPath'), value: snap.paths.dbPath, copy: true },
    {
      key: 'covers',
      label: t('settings.diagnostics.coversPath'),
      value: snap.paths.coversDir,
      copy: true,
    },
    {
      key: 'logs',
      label: t('settings.diagnostics.logsPath'),
      value: snap.paths.logsDir,
      copy: true,
    },
    {
      key: 'data',
      label: t('settings.diagnostics.dataPath'),
      value: snap.paths.dataDir,
      copy: true,
    },
    {
      key: 'version',
      label: t('settings.diagnostics.appVersion'),
      value: t('settings.diagnostics.versionLine', {
        version: snap.version || t('settings.diagnostics.unknown'),
        commit: snap.commit || t('settings.diagnostics.unknown'),
        built: snap.buildDate || t('settings.diagnostics.unknown'),
      }),
      copy: false,
    },
    {
      key: 'os',
      label: t('settings.diagnostics.os'),
      value: `${snap.os}/${snap.arch}`,
      copy: false,
    },
    { key: 'webview', label: t('settings.diagnostics.webView'), value: webView.value, copy: false },
    {
      key: 'dump',
      label: t('settings.diagnostics.dumpVersion'),
      value: snap.inpxVersion || t('settings.diagnostics.unknown'),
      copy: false,
    },
    {
      key: 'works',
      label: t('settings.diagnostics.works'),
      value: formatCount(snap.works, locale.value),
      copy: false,
    },
    {
      key: 'authors',
      label: t('settings.diagnostics.authors'),
      value: formatCount(snap.authors, locale.value),
      copy: false,
    },
  ]
})
</script>

<template>
  <section class="rounded-2xl border border-border bg-card/80 p-6 backdrop-panel">
    <h2 class="font-display text-xl font-medium">{{ t('settings.diagnostics.title') }}</h2>
    <p class="mt-2 text-sm text-muted-foreground">{{ t('settings.diagnostics.lead') }}</p>

    <div v-if="status === 'loading' || status === 'idle'" class="mt-6 grid gap-3" aria-busy="true">
      <span class="sr-only">{{ t('common.loading') }}</span>
      <Skeleton class="h-6 w-2/3" />
      <Skeleton class="h-6 w-1/2" />
      <Skeleton class="h-6 w-3/4" />
      <Skeleton class="h-6 w-2/5" />
    </div>

    <div v-else-if="status === 'error'" class="mt-6">
      <p>{{ diag.errorText() }}</p>
      <Button class="mt-3" @click="diag.load()">{{ t('common.retry') }}</Button>
    </div>

    <div v-else-if="status === 'empty'" class="mt-6">
      <p>{{ t('settings.diagnostics.empty') }}</p>
      <Button class="mt-3" @click="diag.load()">{{ t('common.retry') }}</Button>
    </div>

    <dl v-else class="mt-6 grid gap-4">
      <div v-for="row in rows" :key="row.key" class="grid gap-1">
        <dt class="text-sm text-muted-foreground">{{ row.label }}</dt>
        <dd class="flex flex-wrap items-start gap-2">
          <span class="font-mono text-sm break-all tabular-nums">{{ row.value }}</span>
          <Button
            v-if="row.copy && row.value"
            variant="ghost"
            size="sm"
            @click="copy(row.value, row.key)"
          >
            {{ copiedKey === row.key ? t('common.copied') : t('common.copy') }}
          </Button>
        </dd>
      </div>

      <div class="grid gap-1">
        <dt class="text-sm text-muted-foreground">{{ t('settings.diagnostics.importedAt') }}</dt>
        <dd>
          <Tooltip v-if="importedExact" :delay-duration="200">
            <TooltipTrigger as-child>
              <span class="text-sm tabular-nums">{{
                importedLabel || t('settings.diagnostics.unknown')
              }}</span>
            </TooltipTrigger>
            <TooltipContent>{{ importedExact }}</TooltipContent>
          </Tooltip>
          <span v-else class="text-sm tabular-nums">{{
            importedLabel || t('settings.diagnostics.unknown')
          }}</span>
        </dd>
      </div>

      <div class="grid gap-1">
        <dt class="text-sm text-muted-foreground">{{ t('settings.diagnostics.unnamedGenres') }}</dt>
        <dd class="text-sm">
          <p class="tabular-nums">
            {{
              t('settings.diagnostics.unnamedCount', snapshot?.unnamedGenres ?? 0, {
                n: formatCount(snapshot?.unnamedGenres ?? 0, locale),
              })
            }}
          </p>
          <p class="mt-1 text-muted-foreground">{{ t('settings.diagnostics.unnamedHint') }}</p>
        </dd>
      </div>
    </dl>

    <div class="mt-6 flex flex-wrap gap-2">
      <Button variant="outline" @click="app.openLogsDir()">{{ t('startup.openLogs') }}</Button>
      <Button variant="outline" @click="app.openDataDir()">{{ t('startup.openData') }}</Button>
    </div>

    <div class="mt-8 grid gap-3">
      <h3 class="font-medium">{{ t('settings.diagnostics.archive') }}</h3>
      <p class="text-sm text-muted-foreground">{{ t('settings.diagnostics.archiveHint') }}</p>
      <div v-if="archiveBusy">
        <IndeterminateProgress :label="t('settings.diagnostics.archiveRunning')" />
      </div>
      <Button :disabled="archiveBusy" @click="saveArchive">{{
        t('settings.diagnostics.saveArchive')
      }}</Button>
      <p v-if="archivePath" class="font-mono text-sm break-all">{{ archivePath }}</p>
      <Button v-if="archivePath" variant="ghost" size="sm" @click="showArchive">
        {{ t('settings.diagnostics.showInFolder') }}
      </Button>
    </div>
  </section>

  <IssueReportForm />
</template>
