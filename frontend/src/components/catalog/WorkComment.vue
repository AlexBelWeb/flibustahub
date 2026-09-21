<script setup lang="ts">
import { CircleCheck } from '@lucide/vue'
import { computed, nextTick, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
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

function textOf(value?: string | null) {
  return value ?? ''
}

function hasText(value?: string | null) {
  return textOf(value).trim().length > 0
}

const expanded = ref(hasText(props.comment))
const mode = ref<'edit' | 'preview'>(hasText(props.comment) ? 'preview' : 'edit')
const draft = ref(textOf(props.comment))
const saved = ref(textOf(props.comment))
const saving = ref(false)
const justSaved = ref(false)
const cursor = ref({ start: 0, end: 0 })
const area = ref<{
  el: HTMLTextAreaElement | null
  focus: () => void
  restoreCursor: (start: number, end: number) => void
} | null>(null)

let debounceTimer = 0
let savedTimer = 0

const hasNote = computed(() => hasText(draft.value))
const filledOpen = computed(() => hasNote.value && expanded.value)

watch(
  () => props.workId,
  () => {
    window.clearTimeout(debounceTimer)
    draft.value = textOf(props.comment)
    saved.value = textOf(props.comment)
    expanded.value = hasText(props.comment)
    mode.value = expanded.value ? 'preview' : 'edit'
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

function onOpen(value: boolean) {
  if (!value && hasNote.value) {
    expanded.value = true
    return
  }
  if (expanded.value && !value) {
    void persist()
    mode.value = 'edit'
  }
  expanded.value = value
  if (value && !hasNote.value) {
    mode.value = 'edit'
    void nextTick(() => {
      area.value?.focus()
    })
  }
}

async function startEdit() {
  expanded.value = true
  mode.value = 'edit'
  await nextTick()
  area.value?.focus()
}

function onAreaBlur() {
  rememberCursor()
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
  <Collapsible class="grid gap-2" :open="expanded" @update:open="onOpen">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <h2 v-if="filledOpen" class="font-display text-lg font-medium">
        {{ t('personal.note') }}
      </h2>
      <CollapsibleTrigger v-else as-child>
        <Button
          type="button"
          variant="ghost"
          :class="
            expanded
              ? 'h-auto px-0 py-0 font-display text-lg font-medium text-foreground hover:bg-transparent'
              : 'justify-start font-normal text-muted-foreground'
          "
        >
          {{ expanded ? t('personal.note') : t('personal.addNote') }}
        </Button>
      </CollapsibleTrigger>
      <div v-if="expanded" class="flex items-center gap-2">
        <p
          class="motion-fast flex items-center gap-1 text-xs text-success transition-opacity"
          :class="justSaved ? 'opacity-100' : 'opacity-0'"
        >
          <CircleCheck class="size-3.5" aria-hidden="true" />
          {{ t('personal.saved') }}
        </p>
        <Button
          v-if="hasNote && mode === 'preview'"
          type="button"
          variant="ghost"
          size="sm"
          @click="startEdit"
        >
          {{ t('personal.noteEdit') }}
        </Button>
        <ToggleGroup
          v-else
          :model-value="mode"
          type="single"
          @update:model-value="(v) => onMode(String(v))"
        >
          <ToggleGroupItem value="edit">{{ t('personal.noteEdit') }}</ToggleGroupItem>
          <ToggleGroupItem value="preview">{{ t('personal.notePreview') }}</ToggleGroupItem>
        </ToggleGroup>
      </div>
    </div>
    <CollapsibleContent class="grid gap-2">
      <Textarea
        v-show="mode === 'edit'"
        ref="area"
        :model-value="draft"
        :aria-label="t('personal.note')"
        :aria-describedby="`note-md-hint-${workId}`"
        :placeholder="t('personal.notePlaceholder')"
        @update:model-value="onDraft"
        @blur="onAreaBlur"
      />
      <div
        v-if="mode === 'preview'"
        class="note-preview border-l-2 border-primary pl-3 text-sm leading-relaxed"
        @click="onPreviewClick"
        v-html="renderNote(draft)"
      />
      <p
        v-if="mode === 'edit'"
        :id="`note-md-hint-${workId}`"
        class="text-xs text-muted-foreground"
      >
        {{ t('personal.noteHint') }}
      </p>
    </CollapsibleContent>
  </Collapsible>
</template>
