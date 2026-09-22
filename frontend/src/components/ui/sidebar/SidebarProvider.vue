<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { computed, onMounted, onUnmounted } from 'vue'
import { cn } from '@/lib/utils'
import { provideSidebarContext, SIDEBAR_WIDTH, SIDEBAR_WIDTH_ICON } from './utils'

const props = withDefaults(
  defineProps<{
    open?: boolean
    class?: HTMLAttributes['class']
  }>(),
  { open: true },
)

const emit = defineEmits<{
  'update:open': [open: boolean]
}>()

const open = computed({
  get: () => props.open,
  set: (value) => emit('update:open', value),
})

const state = computed(() => (open.value ? 'expanded' : 'collapsed'))

function setOpen(value: boolean) {
  open.value = value
}

function toggleSidebar() {
  setOpen(!open.value)
}

provideSidebarContext({
  state,
  open,
  setOpen,
  toggleSidebar,
})

function onKey(event: KeyboardEvent) {
  if (event.key.toLowerCase() === 'b' && (event.metaKey || event.ctrlKey)) {
    event.preventDefault()
    toggleSidebar()
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKey)
})
onUnmounted(() => {
  window.removeEventListener('keydown', onKey)
})
</script>

<template>
  <div
    :style="{
      '--sidebar-width': SIDEBAR_WIDTH,
      '--sidebar-width-icon': SIDEBAR_WIDTH_ICON,
    }"
    :class="cn('group/sidebar-wrapper flex h-full min-h-0 min-w-0 flex-1', props.class)"
    data-slot="sidebar-wrapper"
  >
    <slot />
  </div>
</template>
