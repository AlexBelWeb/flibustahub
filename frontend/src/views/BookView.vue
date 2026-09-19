<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import BookDetails from '@/components/catalog/BookDetails.vue'
import { Button } from '@/components/ui/button'
import { routeId } from '@/lib/route-query'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const id = computed(() => routeId(route.params.workId as string))

watch(
  id,
  (workId, prev) => {
    if (workId > 0 && workId !== prev) {
      void window.go.handlers.App.RecordViewed(workId)
    }
  },
  { immediate: true },
)
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col overflow-auto px-6 pt-6 pb-12">
    <p class="mb-6">
      <Button variant="outline" @click="router.back()">{{ t('common.back') }}</Button>
    </p>
    <BookDetails :work-id="id" />
  </div>
</template>
