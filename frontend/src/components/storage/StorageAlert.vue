<script setup lang="ts">
import { TriangleAlert } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { useStorageStore } from '@/stores/storage'

const { t } = useI18n()
const storage = useStorageStore()

function checkAgain() {
  void storage.check(true)
}

function chooseFolder() {
  void storage.chooseFolder(t('onboarding.chooseFolder'))
}
</script>

<template>
  <div v-if="storage.offline" class="border-b border-border">
    <div
      v-if="!storage.alertDismissed"
      class="flex flex-wrap items-start gap-3 bg-warning-quiet px-4 py-3 text-warning"
      role="status"
    >
      <TriangleAlert class="mt-0.5 size-5 shrink-0" aria-hidden="true" />
      <div class="min-w-0 flex-1">
        <p class="font-medium">
          {{ storage.unreachable ? t('storage.unreachableTitle') : t('storage.offlineTitle') }}
        </p>
        <p class="mt-1 text-sm text-muted-foreground">
          {{ storage.unreachable ? t('storage.unreachableBody') : t('storage.offlineBody') }}
        </p>
      </div>
      <div class="flex flex-wrap gap-2">
        <Button size="sm" @click="checkAgain">{{ t('storage.checkAgain') }}</Button>
        <Button size="sm" variant="outline" @click="chooseFolder">{{
          t('storage.repoint')
        }}</Button>
        <Button size="sm" variant="ghost" @click="storage.alertDismissed = true">{{
          t('common.dismiss')
        }}</Button>
      </div>
    </div>
  </div>
</template>
