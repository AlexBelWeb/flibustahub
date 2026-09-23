<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import ListState from '@/components/catalog/ListState.vue'
import WorkBrowser from '@/components/catalog/WorkBrowser.vue'
import { errorMessage } from '@/i18n/errors'
import { parseBackendError } from '@/lib/backend-error'
import { routeId } from '@/lib/route-query'
import { useCatalogStore, type ListStatus } from '@/stores/catalog'
import type { Author } from '@/types/catalog'

const { t } = useI18n()
const route = useRoute()
const catalog = useCatalogStore()
const status = ref<ListStatus>('loading')
const author = ref<Author | null>(null)
const error = ref('')
const id = computed(() => routeId(route.params.authorId as string))

async function load() {
  if (!id.value) {
    status.value = 'missing'
    author.value = null
    return
  }
  status.value = 'loading'
  try {
    author.value = await catalog.getAuthor(id.value)
    status.value = author.value?.id ? 'ready' : 'missing'
  } catch (err) {
    const be = parseBackendError(err)
    if (be.code === 'not_found') {
      status.value = 'missing'
      author.value = null
      return
    }
    error.value = errorMessage(be.code, be.params)
    status.value = 'error'
  }
}

watch(id, () => {
  void load()
})
onMounted(() => {
  void load()
})
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col px-6 py-6">
    <ListState
      :status="status === 'ready' ? 'ready' : status"
      :empty-text="t('common.notFound')"
      :error-text="error || t('list.error')"
      @retry="load"
    >
      <template v-if="author">
        <header class="mb-4">
          <p class="text-sm text-muted-foreground">{{ t('catalog.author') }}</p>
          <h1 class="type-page">{{ author.displayName }}</h1>
          <p class="mt-1 text-sm tabular-nums text-muted-foreground">
            {{ t('authors.works', author.workCount, { n: author.workCount }) }}
          </p>
        </header>
        <WorkBrowser :cache-key="'author:' + author.id" :author-id="author.id" />
      </template>
    </ListState>
  </div>
</template>
