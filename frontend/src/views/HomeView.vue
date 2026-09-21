<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import HomeCarousel from '@/components/catalog/HomeCarousel.vue'
import HomeHero from '@/components/catalog/HomeHero.vue'
import HomeTags from '@/components/catalog/HomeTags.vue'
import ListState from '@/components/catalog/ListState.vue'
import WorkCard from '@/components/catalog/WorkCard.vue'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { CarouselItem } from '@/components/ui/carousel'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { errorMessage } from '@/i18n/errors'
import { parseBackendError } from '@/lib/backend-error'
import { formatCount, formatDate, formatDumpVersionDate, formatRelative } from '@/lib/format'
import { useCatalogStore, type ListStatus } from '@/stores/catalog'
import { useUiStore } from '@/stores/ui'
import type { Work } from '@/types/catalog'

const { t, locale } = useI18n()
const catalog = useCatalogStore()
const ui = useUiStore()

const arrivalsStatus = ref<ListStatus>('idle')
const arrivalsError = ref('')
const arrivals = ref<Work[]>([])
const ratedStatus = ref<ListStatus>('idle')
const ratedError = ref('')
const rated = ref<Work[]>([])
const heroPick = ref<Work | null>(null)
const heroSourcePick = ref('')

const home = computed(() => catalog.home)
const emptyCatalog = computed(() => catalog.homeStatus === 'empty')

const books = computed(() => formatCount(home.value.worksListable, locale.value))
const authors = computed(() => formatCount(home.value.authorsTotal, locale.value))
const series = computed(() => formatCount(home.value.seriesTotal, locale.value))

const importWhen = computed(() => {
  if (home.value.importedAt) {
    return formatRelative(home.value.importedAt, locale.value)
  }
  const fromVersion = formatDumpVersionDate(home.value.inpxVersion, locale.value)
  if (fromVersion) {
    return fromVersion
  }
  return t('home.lastImportNever')
})
const importVersion = computed(() => home.value.inpxVersion || '')
const importExact = computed(() =>
  home.value.importedAt ? formatDate(home.value.importedAt, locale.value) : '',
)

const hero = computed(() => heroPick.value ?? home.value.hero)
const heroSource = computed(() => heroSourcePick.value || home.value.heroSource || '')

const tags = computed(() => {
  const items: Array<{ key: string; to: object; label: string }> = []
  if (home.value.wantToReadCount > 0) {
    items.push({
      key: 'want',
      to: { name: 'books', query: { want: '1', sort: 'wantat' } },
      label: t('home.wantTag', {
        n: formatCount(home.value.wantToReadCount, locale.value),
      }),
    })
  }
  for (const genre of home.value.popularGenres ?? []) {
    items.push({
      key: `g-${genre.id}`,
      to: { name: 'genre', params: { genreId: String(genre.id) } },
      label: genre.nameRu,
    })
  }
  for (const item of home.value.popularSeries ?? []) {
    items.push({
      key: `s-${item.id}`,
      to: { name: 'seriesDetail', params: { seriesId: String(item.id) } },
      label: item.name,
    })
  }
  return items
})

function onHeroReplace(work: Work) {
  heroPick.value = work
  heroSourcePick.value = 'random'
}

function applyCarousels() {
  arrivals.value = home.value.arrivals ?? []
  arrivalsStatus.value = arrivals.value.length ? 'ready' : 'empty'
  arrivalsError.value = ''
  rated.value = home.value.rated ?? []
  ratedStatus.value = rated.value.length ? 'ready' : 'empty'
  ratedError.value = ''
}

async function load() {
  heroPick.value = null
  heroSourcePick.value = ''
  await catalog.loadHome()
  if (catalog.homeStatus === 'ready') {
    applyCarousels()
  }
}

async function retryArrivals() {
  arrivalsStatus.value = 'loading'
  arrivalsError.value = ''
  try {
    const page = await window.go.handlers.App.ListWorks({ sort: 'added', limit: 18 })
    arrivals.value = page.items ?? []
    arrivalsStatus.value = arrivals.value.length ? 'ready' : 'empty'
  } catch (err) {
    const be = parseBackendError(err)
    arrivalsError.value = errorMessage(be.code, be.params)
    arrivalsStatus.value = 'error'
  }
}

async function retryRated() {
  ratedStatus.value = 'loading'
  ratedError.value = ''
  try {
    const page = await window.go.handlers.App.ListWorks({
      sort: 'ratedat',
      rated: true,
      limit: 18,
    })
    rated.value = page.items ?? []
    ratedStatus.value = rated.value.length ? 'ready' : 'empty'
  } catch (err) {
    const be = parseBackendError(err)
    ratedError.value = errorMessage(be.code, be.params)
    ratedStatus.value = 'error'
  }
}

watch(emptyCatalog, (empty) => {
  if (empty && !ui.onboardingDismissed) {
    ui.openOnboarding()
  }
})

onMounted(() => {
  if (catalog.homeStatus === 'idle' || catalog.homeStatus === 'error') {
    void load()
  } else if (catalog.homeStatus === 'ready') {
    applyCarousels()
  } else if (emptyCatalog.value && !ui.onboardingDismissed) {
    ui.openOnboarding()
  }
})

watch(
  () => catalog.home,
  () => {
    if (catalog.homeStatus === 'ready') {
      applyCarousels()
    }
  },
)

watch(
  () => catalog.homeStatus,
  (status) => {
    if (status === 'idle') {
      void load()
    }
  },
)
</script>

<template>
  <div class="flex min-h-0 min-w-0 flex-1 flex-col overflow-x-hidden overflow-y-auto px-6 pt-4">
    <ListState
      class="min-w-0 flex-none"
      :status="catalog.homeStatus"
      :empty-text="t('home.emptyTitle')"
      :error-text="
        catalog.homeError
          ? errorMessage(catalog.homeError.code, catalog.homeError.params)
          : t('list.error')
      "
      @retry="load"
    >
      <template #empty>
        <p class="mt-2 text-muted-foreground">{{ t('home.emptyBody') }}</p>
        <Button class="mt-4 w-fit" @click="ui.openOnboarding()">{{
          t('home.continueSetup')
        }}</Button>
      </template>

      <div class="grid min-w-0 gap-6 pb-12">
        <HomeHero v-if="hero" :work="hero" :source="heroSource" @replace="onHeroReplace" />

        <section class="grid min-w-0 gap-4 sm:grid-cols-2 md:grid-cols-4">
          <Card class="p-6 backdrop-panel">
            <p class="text-sm text-muted-foreground">{{ t('home.books') }}</p>
            <p class="mt-2 font-display text-4xl tabular-nums">{{ books }}</p>
          </Card>
          <Card class="p-6 backdrop-panel">
            <p class="text-sm text-muted-foreground">{{ t('home.authors') }}</p>
            <p class="mt-2 font-display text-4xl tabular-nums">{{ authors }}</p>
          </Card>
          <Card class="p-6 backdrop-panel">
            <p class="text-sm text-muted-foreground">{{ t('home.series') }}</p>
            <p class="mt-2 font-display text-4xl tabular-nums">{{ series }}</p>
          </Card>
          <Card class="min-w-0 p-6 backdrop-panel">
            <p class="text-sm text-muted-foreground">{{ t('home.lastImport') }}</p>
            <Tooltip :disabled="!importExact && !importWhen">
              <TooltipTrigger as-child>
                <div class="mt-2 grid min-w-0 gap-2">
                  <p
                    v-if="importWhen"
                    class="font-display text-2xl leading-none tabular-nums tracking-tight"
                  >
                    {{ importWhen }}
                  </p>
                  <p
                    v-if="importVersion"
                    class="font-display text-2xl leading-none tabular-nums tracking-tight"
                  >
                    {{ importVersion }}
                  </p>
                </div>
              </TooltipTrigger>
              <TooltipContent v-if="importExact">{{ importExact }}</TooltipContent>
            </Tooltip>
          </Card>
        </section>

        <section v-if="tags.length" class="min-w-0">
          <HomeTags :tags="tags" />
        </section>

        <HomeCarousel
          :title="t('home.newArrivals')"
          :more-label="t('home.allArrivals')"
          :more-to="{ name: 'books', query: { sort: 'added' } }"
          :status="arrivalsStatus"
          :empty-text="t('home.arrivalsEmpty')"
          :error-text="arrivalsError || t('list.error')"
          @retry="retryArrivals"
        >
          <CarouselItem
            v-for="work in arrivals"
            :key="work.id"
            class="flex basis-1/2 sm:basis-1/3 md:basis-1/4 lg:basis-1/6"
          >
            <WorkCard
              :work="work"
              class="motion-fast h-full w-full flex-1 transition-transform hover:-translate-y-0.5"
            />
          </CarouselItem>
        </HomeCarousel>

        <HomeCarousel
          :title="t('home.myRatings')"
          :more-label="t('home.allRated')"
          :more-to="{ name: 'books', query: { rated: '1', sort: 'rating' } }"
          :status="ratedStatus"
          :empty-text="t('home.ratedEmpty')"
          :error-text="ratedError || t('list.error')"
          @retry="retryRated"
        >
          <CarouselItem
            v-for="work in rated"
            :key="work.id"
            class="flex basis-1/2 sm:basis-1/3 md:basis-1/4 lg:basis-1/6"
          >
            <WorkCard
              :work="work"
              class="motion-fast h-full w-full flex-1 transition-transform hover:-translate-y-0.5"
            />
          </CarouselItem>
        </HomeCarousel>
      </div>
    </ListState>
  </div>
</template>
