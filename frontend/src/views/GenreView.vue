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
import type { Genre } from '@/types/catalog'

const { t } = useI18n()
const route = useRoute()
const catalog = useCatalogStore()
const status = ref<ListStatus>('loading')
const genre = ref<Genre | null>(null)
const error = ref('')
const id = computed(() => routeId(route.params.genreId as string))

async function load() {
  if (!id.value) {
    status.value = 'missing'
    genre.value = null
    return
  }
  status.value = 'loading'
  try {
    genre.value = await catalog.getGenre(id.value)
    status.value = genre.value?.id ? 'ready' : 'missing'
  } catch (err) {
    const be = parseBackendError(err)
    if (be.code === 'not_found') {
      status.value = 'missing'
      genre.value = null
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
      :status="status"
      :empty-text="t('common.notFound')"
      :error-text="error || t('list.error')"
      @retry="load"
    >
      <template v-if="genre">
        <header class="mb-4">
          <p class="text-sm text-muted-foreground">{{ t('catalog.genre') }}</p>
          <h1 class="type-page">{{ genre.nameRu }}</h1>
          <p class="mt-1 text-sm tabular-nums text-muted-foreground">
            {{ t('genres.works', genre.workCount, { n: genre.workCount }) }}
          </p>
        </header>
        <WorkBrowser :cache-key="'genre:' + genre.id" :genre-id="genre.id" />
      </template>
    </ListState>
  </div>
</template>
