<script setup lang="ts">
import { nextTick, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Textarea } from '@/components/ui/textarea'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { isSafeExternalUrl, renderNote } from '@/lib/markdown'
import { openExternalUrl } from '@/lib/wails-runtime'
import { usePersonalStore } from '@/stores/personal'

const props = defineProps<{
  workId: number
  comment?: string | null
}>()

const { t } = useI18n()
const personal = usePersonalStore()

const mode = ref<'edit' | 'preview'>('edit')
const draft = ref(props.comment ?? '')
const saved = ref(props.comment ?? '')
const saving = ref(false)
const justSaved = ref(false)
const cursor = ref({ start: 0, end: 0 })
const area = ref<{
  el: HTMLTextAreaElement | null
  restoreCursor: (start: number, end: number) => void
} | null>(null)

let debounceTimer = 0
let savedTimer = 0

watch(
  () => props.workId,
  () => {
    window.clearTimeout(debounceTimer)
    draft.value = props.comment ?? ''
    saved.value = props.comment ?? ''
    mode.value = 'edit'
    justSaved.value = false
  },
)

function rememberCursor() {
  const node = area.value?.el
  if (!node) {
    return
  }
  cursor.value = { start: node.selectionStart, end: node.selectionEnd }
}

function onMode(value: string) {
  if (value !== 'edit' && value !== 'preview') {
    return
  }
  if (mode.value === 'edit') {
    rememberCursor()
  }
  mode.value = value
  if (value === 'edit') {
    void nextTick(() => {
      area.value?.restoreCursor(cursor.value.start, cursor.value.end)
    })
  }
}

function scheduleSave() {
  window.clearTimeout(debounceTimer)
  debounceTimer = window.setTimeout(() => {
    void persist()
  }, 1000)
}

function onDraft(value: string) {
  draft.value = value
  if (value === saved.value) {
    window.clearTimeout(debounceTimer)
    return
  }
  scheduleSave()
}

async function persist() {
  if (draft.value === saved.value) {
    return
  }
  if (saving.value) {
    scheduleSave()
    return
  }
  const toSave = draft.value
  saving.value = true
  try {
    await personal.setComment(props.workId, toSave)
    if (draft.value === toSave) {
      saved.value = toSave
      justSaved.value = true
      window.clearTimeout(savedTimer)
      savedTimer = window.setTimeout(() => {
        justSaved.value = false
      }, 1200)
    } else {
      scheduleSave()
    }
  } catch (err) {
    personal.reportSaveError(err, () => {
      void persist()
    })
  } finally {
    saving.value = false
  }
}

function onPreviewClick(event: MouseEvent) {
  const target = event.target
  if (!(target instanceof Element)) {
    return
  }
  const link = target.closest('a')
  if (!link) {
    return
  }
  event.preventDefault()
  const href = link.getAttribute('href') ?? ''
  if (isSafeExternalUrl(href)) {
    openExternalUrl(href)
  }
}

onUnmounted(() => {
  window.clearTimeout(debounceTimer)
  window.clearTimeout(savedTimer)
  if (draft.value !== saved.value) {
    void persist()
  }
})
</script>

<template>
  <section class="grid gap-2">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <h2 class="font-display text-lg font-medium">{{ t('personal.note') }}</h2>
      <div class="flex items-center gap-2">
        <p
          class="text-xs text-muted-foreground transition-opacity duration-200"
          :class="justSaved ? 'opacity-100' : 'opacity-0'"
        >
          {{ t('personal.saved') }}
        </p>
        <ToggleGroup
          :model-value="mode"
          type="single"
          @update:model-value="(v) => onMode(String(v))"
        >
          <ToggleGroupItem value="edit">{{ t('personal.noteEdit') }}</ToggleGroupItem>
          <ToggleGroupItem value="preview">{{ t('personal.notePreview') }}</ToggleGroupItem>
        </ToggleGroup>
      </div>
    </div>
    <Textarea
      v-show="mode === 'edit'"
      ref="area"
      :model-value="draft"
      :aria-label="t('personal.note')"
      @update:model-value="onDraft"
      @blur="rememberCursor"
    />
    <div
      v-show="mode === 'preview'"
      class="note-preview min-h-24 rounded-lg border border-border px-3 py-2 text-sm"
      @click="onPreviewClick"
      v-html="renderNote(draft)"
    />
  </section>
</template>
