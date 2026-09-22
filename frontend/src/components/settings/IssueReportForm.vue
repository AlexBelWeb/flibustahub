<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Progress } from '@/components/ui/progress'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { errorMessage } from '@/i18n/errors'
import { githubIssueURL, mailtoURL } from '@/lib/github-issue'
import { parseBackendError } from '@/lib/backend-error'
import { copyText, openExternalUrl } from '@/lib/wails-runtime'
import { useDiagnosticsStore } from '@/stores/diagnostics'
import { useToastStore } from '@/stores/toast'
import type { IssueKind, IssueReport } from '@/types/diagnostics'

const { t } = useI18n()
const diag = useDiagnosticsStore()
const toast = useToastStore()

const kind = ref<IssueKind>('bug')
const description = ref('')
const report = ref<IssueReport | null>(null)
const body = ref('')
const previewState = ref<'idle' | 'loading' | 'ready' | 'empty' | 'error'>('idle')
const previewError = ref('')
const copied = ref(false)
const saving = ref(false)
const savedPath = ref('')
let timer = 0

const github = computed(() =>
  githubIssueURL(report.value?.title || t('settings.issue.untitled'), body.value),
)
const mailHref = computed(() =>
  mailtoURL(report.value?.title || t('settings.issue.untitled'), body.value),
)

onMounted(() => {
  void rebuild()
})

watch([kind, description], () => {
  window.clearTimeout(timer)
  timer = window.setTimeout(() => {
    void rebuild()
  }, 400)
})

async function rebuild() {
  previewState.value = 'loading'
  previewError.value = ''
  try {
    const next = await diag.buildIssue(kind.value, description.value)
    report.value = next
    body.value = next.body
    previewState.value = next.body ? 'ready' : 'empty'
  } catch (err) {
    const be = parseBackendError(err)
    previewError.value = errorMessage(be.code, be.params)
    report.value = null
    previewState.value = 'error'
  }
}

function openGitHub() {
  if (!github.value.fits) {
    return
  }
  openExternalUrl(github.value.url)
}

function openMail() {
  openExternalUrl(mailHref.value)
}

async function copyReport() {
  const ok = await copyText(body.value)
  if (!ok) {
    toast.pushError(t('settings.issue.copyFailed'))
    return
  }
  copied.value = true
  window.setTimeout(() => {
    copied.value = false
  }, 2000)
}

async function saveMarkdown() {
  saving.value = true
  savedPath.value = ''
  try {
    const result = await diag.saveIssue(t('settings.issue.saveTitle'), body.value)
    if (result?.path) {
      savedPath.value = result.path
    }
  } catch (err) {
    const be = parseBackendError(err)
    toast.pushError(errorMessage(be.code, be.params))
  } finally {
    saving.value = false
  }
}

async function showSaved() {
  if (!savedPath.value) {
    return
  }
  try {
    await window.go.handlers.App.ShowInFolder(savedPath.value)
  } catch (err) {
    const be = parseBackendError(err)
    toast.pushError(errorMessage(be.code, be.params))
  }
}

async function attachArchive() {
  try {
    await diag.saveArchive(t('settings.diagnostics.archiveTitle'))
  } catch (err) {
    const be = parseBackendError(err)
    toast.pushError(errorMessage(be.code, be.params))
  }
}
</script>

<template>
  <section class="mt-8 rounded-2xl border border-border bg-card/80 p-6 backdrop-panel">
    <h2 class="font-display text-xl font-medium">{{ t('settings.issue.title') }}</h2>
    <p class="mt-2 text-sm text-muted-foreground">{{ t('settings.issue.lead') }}</p>

    <div class="mt-5 grid gap-2">
      <h3 class="text-sm font-medium text-muted-foreground">{{ t('settings.issue.kind') }}</h3>
      <ToggleGroup
        type="single"
        :model-value="kind"
        @update:model-value="(value) => value && (kind = value as IssueKind)"
      >
        <ToggleGroupItem value="bug">{{ t('settings.issue.kindBug') }}</ToggleGroupItem>
        <ToggleGroupItem value="idea">{{ t('settings.issue.kindIdea') }}</ToggleGroupItem>
      </ToggleGroup>
    </div>

    <div class="mt-5 grid gap-2">
      <label class="text-sm font-medium text-muted-foreground" for="issue-description">{{
        t('settings.issue.description')
      }}</label>
      <Textarea
        id="issue-description"
        v-model="description"
        class="min-h-28"
        :placeholder="t('settings.issue.descriptionPlaceholder')"
      />
    </div>

    <div class="mt-5 grid gap-2">
      <h3 class="text-sm font-medium">{{ t('settings.issue.preview') }}</h3>
      <p class="text-sm text-muted-foreground">{{ t('settings.issue.previewLead') }}</p>

      <div v-if="previewState === 'loading'" class="grid gap-2" aria-busy="true">
        <span class="sr-only">{{ t('common.loading') }}</span>
        <div class="h-48 animate-pulse rounded-lg bg-muted" />
      </div>
      <div v-else-if="previewState === 'error'" class="rounded-xl border border-border p-4">
        <p>{{ previewError }}</p>
        <Button class="mt-3" @click="rebuild">{{ t('common.retry') }}</Button>
      </div>
      <div v-else-if="previewState === 'empty'" class="rounded-xl border border-border p-4">
        <p>{{ t('settings.issue.previewEmpty') }}</p>
        <Button class="mt-3" @click="rebuild">{{ t('common.retry') }}</Button>
      </div>
      <Textarea
        v-else
        v-model="body"
        class="min-h-64 font-mono text-xs"
        :aria-label="t('settings.issue.preview')"
      />
    </div>

    <div class="mt-5 flex flex-wrap gap-2">
      <Tooltip :disabled="github.fits">
        <TooltipTrigger as-child>
          <span class="inline-flex">
            <Button :disabled="!github.fits || previewState !== 'ready'" @click="openGitHub">
              {{ t('settings.issue.openGitHub') }}
            </Button>
          </span>
        </TooltipTrigger>
        <TooltipContent v-if="!github.fits">{{ t('settings.issue.urlTooLong') }}</TooltipContent>
      </Tooltip>
      <Button variant="outline" :disabled="previewState !== 'ready'" @click="copyReport">
        {{ copied ? t('common.copied') : t('settings.issue.copy') }}
      </Button>
      <Button
        variant="outline"
        :disabled="previewState !== 'ready' || saving"
        @click="saveMarkdown"
      >
        {{ t('settings.issue.save') }}
      </Button>
    </div>
    <p v-if="!github.fits" class="mt-2 text-sm text-muted-foreground">
      {{ t('settings.issue.urlTooLong') }}
    </p>
    <p v-if="savedPath" class="mt-2 font-mono text-sm break-all">{{ savedPath }}</p>
    <Button v-if="savedPath" class="mt-2" variant="ghost" size="sm" @click="showSaved">
      {{ t('settings.diagnostics.showInFolder') }}
    </Button>

    <div class="mt-6 grid gap-3">
      <p class="text-sm text-muted-foreground">{{ t('settings.diagnostics.archiveHint') }}</p>
      <div v-if="diag.archiveBusy">
        <Progress :model-value="null" :label="t('settings.diagnostics.archiveRunning')" />
      </div>
      <Button variant="outline" :disabled="diag.archiveBusy" @click="attachArchive">
        {{ t('settings.issue.attachArchive') }}
      </Button>
      <Button variant="ghost" @click="openMail">{{ t('settings.issue.mailto') }}</Button>
    </div>
  </section>
</template>
