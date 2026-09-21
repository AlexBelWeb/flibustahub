<script setup lang="ts">
import { CircleAlert, CircleCheck, TriangleAlert } from '@lucide/vue'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { formatCount, formatDuration } from '@/lib/format'
import { emptyNotes, type ImportReport } from '@/types/import'

const props = defineProps<{
  report: ImportReport
  compact?: boolean
}>()

const { t, locale } = useI18n()
const copied = ref(false)

const loc = computed(() => locale.value)
const notes = computed(() => {
  const n = props.report.notes
  if (!n) {
    return emptyNotes()
  }
  return {
    ...emptyNotes(),
    ...n,
    missingArchives: n.missingArchives ?? [],
    unnamedGenres: n.unnamedGenres ?? [],
    encodings: n.encodings ?? { utf8: 0, cp1251: 0 },
  }
})
const mixedEncodings = computed(
  () => notes.value.encodings.utf8 > 0 && notes.value.encodings.cp1251 > 0,
)
const missingList = computed(() => (notes.value.missingArchives ?? []).join('\n'))

const stats = computed(() => [
  { label: t('import.recordsSeen'), value: props.report.recordsSeen },
  { label: t('import.worksAdded'), value: props.report.worksAdded },
  { label: t('import.editionsAdded'), value: props.report.editionsAdded },
  { label: t('import.editionsUpdated'), value: props.report.editionsUpdated },
  { label: t('import.editionsDeactivated'), value: props.report.editionsDeactivated },
  { label: t('import.libidCollisions'), value: props.report.libidCollisions },
])

const phaseRows = computed(() => {
  const ms = notes.value.phasesMs ?? {}
  return [
    { label: t('import.phases.backup'), ms: ms.backup },
    { label: t('import.phases.reading'), ms: ms.reading },
    { label: t('import.phases.records'), ms: ms.records },
    { label: t('import.phases.analyze'), ms: ms.analyze },
    { label: t('import.phases.fts'), ms: ms.fts },
    { label: t('import.phases.warmup'), ms: ms.warmup },
  ].filter((row) => typeof row.ms === 'number')
})

function count(n: number) {
  return formatCount(n, loc.value)
}

function statusLabel(status: string) {
  if (status === 'cancelled') {
    return t('import.statusCancelled')
  }
  if (status === 'failed') {
    return t('import.statusFailed')
  }
  return t('import.statusDone')
}

function phaseTime(ms: number) {
  if (ms < 1000) {
    return t('import.phaseMs', { n: count(ms) })
  }
  return t('import.phaseSec', { n: formatDuration(ms, loc.value) })
}

async function copyMissing() {
  if (!missingList.value) {
    return
  }
  try {
    await navigator.clipboard.writeText(missingList.value)
    copied.value = true
    window.setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch {
    copied.value = false
  }
}
</script>

<template>
  <div class="grid gap-5">
    <div class="flex flex-wrap items-baseline justify-between gap-2">
      <p
        class="flex items-center gap-2 font-medium"
        :class="
          report.status === 'failed'
            ? 'text-destructive'
            : report.status === 'cancelled'
              ? 'text-warning'
              : 'text-success'
        "
      >
        <CircleAlert v-if="report.status === 'failed'" class="size-4" aria-hidden="true" />
        <TriangleAlert
          v-else-if="report.status === 'cancelled'"
          class="size-4"
          aria-hidden="true"
        />
        <CircleCheck v-else class="size-4" aria-hidden="true" />
        {{ statusLabel(report.status) }}
      </p>
      <p v-if="report.inpxVersion" class="text-sm text-muted-foreground tabular-nums">
        {{ t('import.lastVersion', { version: report.inpxVersion }) }}
      </p>
    </div>

    <dl class="grid gap-2 sm:grid-cols-2">
      <div v-for="row in stats" :key="row.label" class="flex items-baseline justify-between gap-3">
        <dt class="text-sm text-muted-foreground">{{ row.label }}</dt>
        <dd class="tabular-nums">{{ count(row.value) }}</dd>
      </div>
    </dl>

    <section
      class="rounded-xl border p-4"
      :class="
        notes.missingArchivesTotal > 0
          ? 'border-transparent bg-warning-quiet text-warning'
          : 'border-transparent bg-success-quiet text-success'
      "
    >
      <h3 class="font-display text-lg font-medium">{{ t('import.missingTitle') }}</h3>
      <p v-if="notes.missingArchivesTotal === 0" class="mt-2 text-sm text-muted-foreground">
        {{ t('import.missingOk') }}
      </p>
      <template v-else>
        <p class="mt-2 tabular-nums">
          {{
            t('import.missingCount', notes.missingArchivesTotal, {
              n: count(notes.missingArchivesTotal),
            })
          }}
        </p>
        <p class="mt-1 text-sm text-muted-foreground">{{ t('import.missingHint') }}</p>
        <pre
          class="mt-3 max-h-48 overflow-auto rounded-lg bg-background p-3 font-mono text-xs leading-relaxed select-text"
          >{{ missingList }}</pre>
        <Button class="mt-3" size="sm" variant="outline" @click="copyMissing">
          {{ copied ? t('common.copied') : t('common.copy') }}
        </Button>
      </template>
    </section>

    <template v-if="!compact">
      <section class="grid gap-2">
        <h3 class="font-medium">{{ t('import.unnamedTitle') }}</h3>
        <p v-if="notes.unnamedGenresTotal === 0" class="text-sm text-muted-foreground">
          {{ t('import.unnamedOk') }}
        </p>
        <template v-else>
          <p class="tabular-nums">
            {{
              t('import.unnamedCount', notes.unnamedGenresTotal, {
                n: count(notes.unnamedGenresTotal),
              })
            }}
          </p>
          <p class="font-mono text-sm break-all">{{ notes.unnamedGenres.join(', ') }}</p>
        </template>
      </section>

      <p v-if="notes.genreNamesMapped > 0" class="tabular-nums">
        {{ t('import.mappedTitle') }}:
        {{ t('import.mappedCount', notes.genreNamesMapped, { n: count(notes.genreNamesMapped) }) }}
      </p>

      <section class="grid gap-2">
        <h3 class="font-medium">{{ t('import.skippedTitle') }}</h3>
        <dl class="grid gap-1 text-sm">
          <div class="flex justify-between gap-3">
            <dt class="text-muted-foreground">{{ t('import.skippedMalformed') }}</dt>
            <dd class="tabular-nums">{{ count(notes.skippedMalformed) }}</dd>
          </div>
          <div class="flex justify-between gap-3">
            <dt class="text-muted-foreground">{{ t('import.skippedNoLibid') }}</dt>
            <dd class="tabular-nums">{{ count(notes.skippedNoLibid) }}</dd>
          </div>
        </dl>
      </section>

      <section class="grid gap-2">
        <h3 class="font-medium">{{ t('import.encodingsTitle') }}</h3>
        <p v-if="mixedEncodings" class="flex items-center gap-2 text-sm text-warning">
          <TriangleAlert class="size-4" aria-hidden="true" />
          {{ t('import.encodingsMixed') }}
        </p>
        <dl class="grid gap-1 text-sm">
          <div class="flex justify-between gap-3">
            <dt class="text-muted-foreground">{{ t('import.encodingsUtf8') }}</dt>
            <dd class="tabular-nums">{{ count(notes.encodings.utf8) }}</dd>
          </div>
          <div class="flex justify-between gap-3">
            <dt class="text-muted-foreground">{{ t('import.encodingsCp1251') }}</dt>
            <dd class="tabular-nums">{{ count(notes.encodings.cp1251) }}</dd>
          </div>
          <div v-if="notes.encodings.versionInfo" class="flex justify-between gap-3">
            <dt class="font-mono text-muted-foreground">{{ t('import.encodingsVersion') }}</dt>
            <dd>{{ notes.encodings.versionInfo }}</dd>
          </div>
          <div v-if="notes.encodings.collectionInfo" class="flex justify-between gap-3">
            <dt class="font-mono text-muted-foreground">{{ t('import.encodingsCollection') }}</dt>
            <dd>{{ notes.encodings.collectionInfo }}</dd>
          </div>
        </dl>
      </section>

      <section v-if="phaseRows.length" class="grid gap-2">
        <h3 class="font-medium">{{ t('import.phasesTitle') }}</h3>
        <dl class="grid gap-1 text-sm">
          <div v-for="row in phaseRows" :key="row.label" class="flex justify-between gap-3">
            <dt class="text-muted-foreground">{{ row.label }}</dt>
            <dd class="tabular-nums">{{ phaseTime(row.ms ?? 0) }}</dd>
          </div>
        </dl>
      </section>
    </template>
  </div>
</template>
