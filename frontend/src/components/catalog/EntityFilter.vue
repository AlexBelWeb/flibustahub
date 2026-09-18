<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
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
const options = ref<Array<{ id: number; label: string }>>([])
let timer = 0

const placeholder = computed(() => {
  if (props.kind === 'author') {
    return t('authors.search')
  }
  if (props.kind === 'series') {
    return t('series.search')
  }
  return t('genres.search')
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

async function resolveSelected() {
  if (!props.modelValue) {
    selectedLabel.value = ''
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
  } catch {
    selectedLabel.value = ''
  }
}

async function loadOptions(q: string) {
  if (props.kind === 'genre') {
    const items = (await window.go.handlers.App.ListGenres(q)) ?? []
    options.value = items.map((item) => ({ id: item.id, label: item.nameRu }))
    return
  }
  if (props.kind === 'author') {
    const page = await window.go.handlers.App.ListAuthors({ query: q, limit: 20 })
    options.value = (page.items ?? []).map((item) => ({ id: item.id, label: item.displayName }))
    return
  }
  const page = await window.go.handlers.App.ListSeries({ query: q, limit: 20 })
  options.value = (page.items ?? []).map((item) => ({ id: item.id, label: item.name }))
}

function onQuery(value: string) {
  query.value = value
  window.clearTimeout(timer)
  timer = window.setTimeout(() => {
    void loadOptions(value)
  }, 250)
}

function pick(id: number, label: string) {
  selectedLabel.value = label
  emit('update:modelValue', id)
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
</script>

<template>
  <div class="grid gap-2">
    <p class="text-sm text-muted-foreground">{{ kindLabel }}</p>
    <Popover v-model:open="open">
      <PopoverTrigger as-child>
        <Button variant="outline" class="w-full justify-start font-normal">
          {{ selectedLabel || placeholder }}
        </Button>
      </PopoverTrigger>
      <PopoverContent class="w-72 p-0">
        <Command :ignore-filter="true" :open="true">
          <CommandInput
            :model-value="query"
            :placeholder="placeholder"
            @update:model-value="onQuery"
          />
          <CommandList>
            <CommandEmpty>{{ t('list.emptySearch') }}</CommandEmpty>
            <CommandItem
              v-for="item in options"
              :key="item.id"
              :value="kind + ':' + item.id"
              @select="pick(item.id, item.label)"
            >
              {{ item.label }}
            </CommandItem>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  </div>
</template>
