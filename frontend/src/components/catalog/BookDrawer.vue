<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import BookDetails from '@/components/catalog/BookDetails.vue'
import { Sheet, SheetContent, SheetDescription, SheetTitle } from '@/components/ui/sheet'
import { workIdFromQuery } from '@/lib/work-route'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const workId = computed(() => workIdFromQuery(route.query))
const open = computed(() => workId.value > 0)

function onOpen(next: boolean) {
  if (!next && workId.value) {
    void router.back()
  }
}

watch(
  workId,
  (id, prev) => {
    if (id > 0 && id !== prev) {
      void window.go.handlers.App.RecordViewed(id)
    }
  },
  { immediate: true },
)
</script>

<template>
  <Sheet :open="open" modal @update:open="onOpen">
    <SheetContent
      side="right"
      hide-close
      class="flex h-full w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-xl"
    >
      <SheetTitle class="sr-only">{{ t('book.drawerTitle') }}</SheetTitle>
      <SheetDescription class="sr-only">{{ t('book.drawerLead') }}</SheetDescription>
      <BookDetails v-if="workId" :work-id="workId" drawer />
    </SheetContent>
  </Sheet>
</template>
