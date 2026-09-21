<script setup lang="ts">
import { TriangleAlert } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { useImportStore } from '@/stores/import'
import { useStorageStore } from '@/stores/storage'

const { t } = useI18n()
const storage = useStorageStore()
const imp = useImportStore()

async function updateCatalog() {
  const offer = storage.dumpOffer
  if (!offer) {
    return
  }
  await window.go.handlers.App.SetINPXPath(offer.path)
  await imp.loadCard()
  imp.requestStart()
}

function dismiss() {
  void storage.dismissDump()
}
</script>

<template>
  <div
    v-if="storage.dumpOffer && storage.available"
    class="flex flex-wrap items-center gap-3 border-b border-border bg-warning-quiet px-4 py-2 text-warning"
    role="status"
  >
    <TriangleAlert class="size-4 shrink-0" aria-hidden="true" />
    <p class="min-w-0 flex-1 text-sm">
      {{ t('storage.dumpNewer', { name: storage.dumpOffer.name }) }}
    </p>
    <Button size="sm" @click="updateCatalog">{{ t('storage.dumpUpdate') }}</Button>
    <Button size="sm" variant="ghost" @click="dismiss">{{ t('common.dismiss') }}</Button>
  </div>
</template>
