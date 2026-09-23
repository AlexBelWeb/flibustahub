<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'
import { useSidebar } from './utils'

withDefaults(
  defineProps<{
    side?: 'left' | 'right'
    variant?: 'sidebar' | 'floating' | 'inset'
    collapsible?: 'offcanvas' | 'icon' | 'none'
    class?: HTMLAttributes['class']
  }>(),
  {
    side: 'left',
    variant: 'sidebar',
    collapsible: 'icon',
  },
)

const { state } = useSidebar()
</script>

<template>
  <div
    v-if="collapsible === 'none'"
    :class="
      cn(
        'flex h-full w-(--sidebar-width) flex-col bg-sidebar text-sidebar-foreground',
        $props.class,
      )
    "
  >
    <slot />
  </div>
  <div
    v-else
    class="group peer text-sidebar-foreground"
    :data-state="state"
    :data-collapsible="state === 'collapsed' ? collapsible : ''"
    :data-variant="variant"
    :data-side="side"
  >
    <div
      :class="
        cn(
          'sidebar-width relative h-full bg-transparent',
          'group-data-[collapsible=icon]:w-(--sidebar-width-icon)',
          'w-(--sidebar-width)',
        )
      "
    />
    <div
      :class="
        cn(
          'sidebar-width fixed inset-y-0 z-10 flex h-full w-(--sidebar-width)',
          'group-data-[collapsible=icon]:w-(--sidebar-width-icon)',
          side === 'left'
            ? 'left-0 border-r border-sidebar-border'
            : 'right-0 border-l border-sidebar-border',
          $props.class,
        )
      "
    >
      <div
        data-sidebar="sidebar"
        class="flex h-full w-full flex-col bg-sidebar text-sidebar-foreground backdrop-panel"
      >
        <slot />
      </div>
    </div>
  </div>
</template>
