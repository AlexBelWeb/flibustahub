<script setup lang="ts">
import { PanelLeft } from '@lucide/vue'
import type { HTMLAttributes } from 'vue'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { useSidebar } from './utils'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()

const { toggleSidebar } = useSidebar()
</script>

<template>
  <Button
    data-sidebar="trigger"
    variant="ghost"
    size="icon"
    :class="cn(props.class)"
    @click="toggleSidebar"
  >
    <PanelLeft class="size-4" />
    <span class="sr-only">{{ $t('nav.collapse') }}</span>
  </Button>
</template>
