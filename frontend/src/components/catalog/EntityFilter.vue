<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import {
  ComboboxAnchor,
  ComboboxCancel,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxRoot,
  ComboboxTrigger,
} from 'reka-ui'
import { useCatalogStore } from '@/stores/catalog'

const props = defineProps<{
  kind: 'author' | 'genre' | 'series'
  modelValue: number
}>()

const emit = defineEmits<{
  'update:modelValue': [id: number]
}>()

const { t } = useI18n()
const catalog = useCatalogStore()
const open = ref(false)
const query = ref('')
const selectedLabel = ref('')
const unresolved = ref(false)
const options = ref<Array<{ id: number; label: string }>>([])
const status = ref<'idle' | 'loading' | 'empty' | 'ready' | 'error'>('idle')
let timer = 0
let request = 0

const anyLabel = computed(() => {
  if (props.kind === 'author') {
    return t('catalog.filterAnyAuthor')
  }
  if (props.kind === 'series') {
    return t('catalog.filterAnySeries')
  }
  return t('catalog.filterAnyGenre')
})

const kindLabel = computed(() => {
  if (props.kind === 'author') {
    return t('catalog.author')
  }
  if (props.kind === 'series') {
    return t('catalog.series')
  }
  return t('catalog.genre')
})

const selected = computed(() => (props.modelValue > 0 ? props.modelValue : undefined))

function displayValue() {
  if (!props.modelValue) {
    return ''
  }
  return selectedLabel.value || String(props.modelValue)
}

async function resolveSelected() {
  if (!props.modelValue) {
    selectedLabel.value = ''
    unresolved.value = false
    return
  }
  try {
    if (props.kind === 'author') {
      const item = await catalog.getAuthor(props.modelValue)
      selectedLabel.value = item?.displayName ?? ''
    } else if (props.kind === 'series') {
      const item = await catalog.getSeries(props.modelValue)
      selectedLabel.value = item?.name ?? ''
    } else {
      const item = await catalog.getGenre(props.modelValue)
      selectedLabel.value = item?.nameRu ?? ''
    }
    unresolved.value = selectedLabel.value === ''
    if (unresolved.value) {
      selectedLabel.value = String(props.modelValue)
    }
  } catch {
    selectedLabel.value = String(props.modelValue)
    unresolved.value = true
  }
}

async function loadOptions(q: string) {
  const current = ++request
  status.value = 'loading'
  try {
    if (props.kind === 'genre') {
      const items = (await window.go.handlers.App.ListGenres(q)) ?? []
      if (current !== request) {
        return
      }
      options.value = items.map((item) => ({ id: item.id, label: item.nameRu }))
    } else if (props.kind === 'author') {
      const page = await window.go.handlers.App.ListAuthors({ query: q, limit: 20 })
      if (current !== request) {
        return
      }
      options.value = (page.items ?? []).map((item) => ({ id: item.id, label: item.displayName }))
    } else {
      const page = await window.go.handlers.App.ListSeries({ query: q, limit: 20 })
      if (current !== request) {
        return
      }
      options.value = (page.items ?? []).map((item) => ({ id: item.id, label: item.name }))
    }
    status.value = options.value.length ? 'ready' : 'empty'
  } catch {
    if (current !== request) {
      return
    }
    options.value = []
    status.value = 'error'
  }
}

function onQuery(value: string) {
  query.value = value
  status.value = 'loading'
  window.clearTimeout(timer)
  timer = window.setTimeout(() => {
    void loadOptions(value)
  }, 250)
}

function onModel(value: unknown) {
  const id = typeof value === 'number' ? value : 0
  if (!id) {
    clear()
    return
  }
  const item = options.value.find((entry) => entry.id === id)
  selectedLabel.value = item?.label ?? String(id)
  unresolved.value = !item
  emit('update:modelValue', id)
  open.value = false
}

function clear() {
  selectedLabel.value = ''
  unresolved.value = false
  emit('update:modelValue', 0)
  open.value = false
}

watch(
  () => props.modelValue,
  () => {
    void resolveSelected()
  },
  { immediate: true },
)

watch(open, (value) => {
  if (value) {
    query.value = ''
    void loadOptions('')
  }
})

onUnmounted(() => {
  window.clearTimeout(timer)
  request += 1
})
</script>

<template>
  <div class="grid gap-2">
    <p class="text-sm text-muted-foreground">{{ kindLabel }}</p>
    <ComboboxRoot
      :open="open"
      :model-value="selected"
      :ignore-filter="true"
      :reset-model-value-on-clear="true"
      @update:open="open = $event"
      @update:model-value="onModel"
    >
      <ComboboxAnchor class="flex items-center gap-1">
        <ComboboxInput
          :display-value="displayValue"
          :placeholder="anyLabel"
          class="h-9 min-w-0 flex-1 rounded-lg border border-input bg-background px-3 text-sm text-foreground outline-none placeholder:text-muted-foreground focus-visible:ring-1 focus-visible:ring-ring"
          @update:model-value="onQuery"
        />
        <ComboboxCancel v-if="modelValue" as-child>
          <Button type="button" variant="ghost" size="sm">
            {{ t('catalog.removeFilter') }}
          </Button>
        </ComboboxCancel>
        <ComboboxTrigger
          class="inline-flex size-9 items-center justify-center rounded-lg border border-input"
          :aria-label="kindLabel"
        />
      </ComboboxAnchor>
      <p v-if="unresolved" class="text-xs text-muted-foreground">
        {{ t('catalog.filterUnreadable', { id: modelValue }) }}
      </p>
      <ComboboxContent
        position="popper"
        class="z-50 max-h-72 w-(--reka-popper-anchor-width) overflow-auto rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-md"
      >
        <p v-if="status === 'loading'" class="px-2 py-1.5 text-sm text-muted-foreground">
          {{ t('common.loading') }}
        </p>
        <div v-else-if="status === 'error'" class="grid gap-2 px-2 py-1.5">
          <p class="text-sm text-destructive">{{ t('list.error') }}</p>
          <Button type="button" size="sm" variant="outline" @click="loadOptions(query)">
            {{ t('common.retry') }}
          </Button>
        </div>
        <ComboboxEmpty v-else-if="status === 'empty'" class="px-2 py-1.5 text-sm">
          {{ t('list.emptySearch') }}
        </ComboboxEmpty>
        <ComboboxItem
          v-for="item in status === 'ready' ? options : []"
          :key="item.id"
          :value="item.id"
          :text-value="item.label"
          class="cursor-pointer rounded-sm px-2 py-1.5 text-sm data-[highlighted]:bg-accent"
        >
          {{ item.label }}
        </ComboboxItem>
      </ComboboxContent>
    </ComboboxRoot>
  </div>
</template>
