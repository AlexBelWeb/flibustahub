<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

const props = defineProps<{
  status: string
  emptyText: string
  errorText: string
  class?: HTMLAttributes['class']
}>()

const emit = defineEmits<{
  retry: []
}>()
</script>

<template>
  <div :class="cn('flex min-h-0 min-w-0 flex-1 flex-col', props.class)">
    <div v-if="status === 'loading'" class="grid gap-3" aria-busy="true">
      <span class="sr-only">{{ $t('common.loading') }}</span>
      <Skeleton class="h-40 rounded-xl" />
      <Skeleton class="h-40 rounded-xl" />
      <Skeleton class="h-40 rounded-xl" />
    </div>
    <div v-else-if="status === 'error'" class="rounded-2xl border border-border bg-card p-6">
      <p class="mb-4">{{ errorText }}</p>
      <Button @click="emit('retry')">{{ $t('common.retry') }}</Button>
    </div>
    <div
      v-else-if="status === 'empty' || status === 'missing'"
      class="rounded-2xl border border-border bg-card p-6"
    >
      <p>{{ emptyText }}</p>
      <slot name="empty" />
    </div>
    <div v-else class="flex min-h-0 min-w-0 flex-1 flex-col">
      <slot />
    </div>
  </div>
</template>
