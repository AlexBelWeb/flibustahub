<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import ListState from '@/components/catalog/ListState.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { errorMessage } from '@/i18n/errors'
import { parseBackendError } from '@/lib/backend-error'
import { formatBytes } from '@/lib/format'
import { routeId } from '@/lib/route-query'
import { authorParts, coverHue, isBlankTitle } from '@/lib/work'
import { useCatalogStore, type ListStatus } from '@/stores/catalog'
import type { Work } from '@/types/catalog'

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const catalog = useCatalogStore()

const status = ref<ListStatus>('loading')
const work = ref<Work | null>(null)
const error = ref('')

const id = computed(() => routeId(route.params.workId as string))
const title = computed(() =>
  work.value && !isBlankTitle(work.value.title) ? work.value.title : t('catalog.untitled'),
)
const authors = computed(() => authorParts(work.value?.authorsText).join(', '))
const hue = computed(() => coverHue(work.value?.workKey || String(work.value?.id ?? 0)))
const size = computed(() => (work.value?.size ? formatBytes(work.value.size, locale.value) : ''))

async function load() {
  if (!id.value) {
    status.value = 'missing'
    work.value = null
    return
  }
  status.value = 'loading'
  error.value = ''
  try {
    const next = await catalog.getWork(id.value)
    work.value = next
    status.value = next?.id ? 'ready' : 'missing'
  } catch (err) {
    const be = parseBackendError(err)
    if (be.code === 'not_found') {
      status.value = 'missing'
      work.value = null
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
  <div class="flex min-h-0 flex-1 flex-col overflow-auto px-6 py-6">
    <p class="mb-6">
      <Button variant="outline" @click="router.back()">{{ t('common.back') }}</Button>
    </p>
    <ListState
      :status="status"
      :empty-text="t('common.notFound')"
      :error-text="error || t('list.error')"
      @retry="load"
    >
      <article v-if="work" class="grid max-w-3xl gap-6">
        <div
          class="flex min-h-64 flex-col justify-end rounded-2xl p-6 text-primary-foreground"
          :style="{ background: `hsl(${hue} 32% var(--cover-l, 28%))` }"
        >
          <h1 class="font-display text-4xl font-semibold">{{ title }}</h1>
          <p v-if="authors" class="mt-3 text-lg opacity-90">{{ authors }}</p>
        </div>
        <dl class="grid gap-3 text-sm">
          <div v-if="work.series" class="grid gap-1">
            <dt class="text-muted-foreground">{{ t('book.series') }}</dt>
            <dd>
              {{ work.series }}
              <span v-if="work.seriesNo">{{ t('catalog.seriesNo', { n: work.seriesNo }) }}</span>
            </dd>
          </div>
          <div v-if="work.lang" class="grid gap-1">
            <dt class="text-muted-foreground">{{ t('book.lang') }}</dt>
            <dd>{{ work.lang }}</dd>
          </div>
          <div v-if="size" class="grid gap-1">
            <dt class="text-muted-foreground">{{ t('book.size') }}</dt>
            <dd class="tabular-nums">{{ size }}</dd>
          </div>
        </dl>
        <div class="flex flex-wrap gap-2">
          <Badge
            v-if="work.editionCount > 1"
            variant="secondary"
            class="px-3 py-1 text-sm tabular-nums"
          >
            {{ t('catalog.files', work.editionCount, { n: work.editionCount }) }}
          </Badge>
          <Badge v-if="!work.hasFile" variant="muted" class="px-3 py-1 text-sm">
            {{ t('catalog.ghost') }}
          </Badge>
          <Badge v-if="work.rating" class="px-3 py-1 text-sm tabular-nums">
            {{ work.rating }}
          </Badge>
          <Badge v-if="work.librate" variant="outline" class="px-3 py-1 text-sm tabular-nums">
            {{ work.librate }}
          </Badge>
        </div>
      </article>
    </ListState>
  </div>
</template>
