<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import BookCover from '@/components/catalog/BookCover.vue'
import HomeCarousel from '@/components/catalog/HomeCarousel.vue'
import ListState from '@/components/catalog/ListState.vue'
import WorkCard from '@/components/catalog/WorkCard.vue'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { CarouselItem } from '@/components/ui/carousel'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { errorMessage } from '@/i18n/errors'
import { parseBackendError } from '@/lib/backend-error'
import { formatCount, formatDate, formatRelative } from '@/lib/format'
import { authorParts, isBlankTitle, isUnknownAuthor } from '@/lib/work'
import { withWorkQuery } from '@/lib/work-route'
import { useCatalogStore, type ListStatus } from '@/stores/catalog'
import { useUiStore } from '@/stores/ui'
import type { Work } from '@/types/catalog'

const { t, locale } = useI18n()
const catalog = useCatalogStore()
const ui = useUiStore()
const route = useRoute()

const arrivalsStatus = ref<ListStatus>('idle')
const arrivalsError = ref('')
const arrivals = ref<Work[]>([])
const ratedStatus = ref<ListStatus>('idle')
const ratedError = ref('')
const rated = ref<Work[]>([])

const home = computed(() => catalog.home)
const emptyCatalog = computed(() => catalog.homeStatus === 'empty')

const books = computed(() => formatCount(home.value.worksListable, locale.value))
const authors = computed(() => formatCount(home.value.authorsTotal, locale.value))
const series = computed(() => formatCount(home.value.seriesTotal, locale.value))

const importLine = computed(() => {
  if (!home.value.importedAt && !home.value.inpxVersion) {
    return t('home.lastImportNever')
  }
  const version = home.value.inpxVersion
    ? t('home.lastImportVersion', { version: home.value.inpxVersion })
    : ''
  const when = home.value.importedAt ? formatRelative(home.value.importedAt, locale.value) : ''
  return [when, version].filter(Boolean).join(' · ')
})

const importExact = computed(() =>
  home.value.importedAt ? formatDate(home.value.importedAt, locale.value) : '',
)

const hero = computed(() => home.value.hero)
const heroTitle = computed(() =>
  hero.value && !isBlankTitle(hero.value.title) ? hero.value.title : t('catalog.untitled'),
)
const heroAuthors = computed(() => {
  if (!hero.value) {
    return ''
  }
  const named = authorParts(hero.value.authorsText).filter((name) => !isUnknownAuthor(name))
  return named.length ? named.join(', ') : t('catalog.unknownAuthor')
})
const heroLead = computed(() =>
  home.value.heroSource === 'random' ? t('home.randomBook') : t('home.returnToBook'),
)

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

function applyCarousels() {
  arrivals.value = home.value.arrivals ?? []
  arrivalsStatus.value = arrivals.value.length ? 'ready' : 'empty'
  arrivalsError.value = ''
  rated.value = home.value.rated ?? []
  ratedStatus.value = rated.value.length ? 'ready' : 'empty'
  ratedError.value = ''
}

async function load() {
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
  <div class="flex min-h-0 flex-1 flex-col overflow-auto px-6 pt-8">
    <ListState
      class="flex-none"
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

      <div class="grid gap-8 pb-12">
        <section
          v-if="hero"
          class="relative overflow-hidden rounded-3xl border border-border min-h-56"
        >
          <BookCover
            :work="hero"
            prio="open"
            :observe="false"
            fit="cover"
            class="absolute inset-0 h-full w-full"
          />
          <div
            class="relative z-10 flex flex-col gap-6 p-6 sm:flex-row sm:items-end sm:justify-between sm:p-8 backdrop-panel"
          >
            <div class="grid min-w-0 max-w-xl gap-3">
              <p class="text-sm text-muted-foreground">{{ heroLead }}</p>
              <h2 class="font-display text-3xl font-semibold">{{ heroTitle }}</h2>
              <p class="text-muted-foreground">{{ heroAuthors }}</p>
              <Button class="w-fit" size="lg" as-child>
                <RouterLink :to="{ query: withWorkQuery(route.query, hero.id) }">
                  {{ t('home.open') }}
                </RouterLink>
              </Button>
            </div>
            <BookCover
              :work="hero"
              prio="open"
              :observe="false"
              fit="contain"
              class="hidden aspect-[2/3] w-32 shrink-0 overflow-hidden rounded-xl sm:block"
            />
          </div>
        </section>

        <section class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
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
          <Card class="p-6 backdrop-panel">
            <p class="text-sm text-muted-foreground">{{ t('home.lastImport') }}</p>
            <Tooltip :disabled="!importExact">
              <TooltipTrigger as-child>
                <p class="mt-2 text-lg">{{ importLine }}</p>
              </TooltipTrigger>
              <TooltipContent v-if="importExact">{{ importExact }}</TooltipContent>
            </Tooltip>
          </Card>
        </section>

        <section v-if="tags.length" class="grid gap-3">
          <div class="flex gap-2 overflow-x-auto pb-1">
            <RouterLink
              v-for="tag in tags"
              :key="tag.key"
              :to="tag.to"
              class="shrink-0 rounded-full border border-border bg-card px-3 py-1 text-sm hover:bg-accent"
            >
              {{ tag.label }}
            </RouterLink>
          </div>
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
            class="basis-1/2 sm:basis-1/3 md:basis-1/4 lg:basis-1/6"
          >
            <WorkCard
              :work="work"
              class="h-full transition-transform duration-200 hover:-translate-y-0.5"
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
            class="basis-1/2 sm:basis-1/3 md:basis-1/4 lg:basis-1/6"
          >
            <WorkCard
              :work="work"
              class="h-full transition-transform duration-200 hover:-translate-y-0.5"
            />
          </CarouselItem>
        </HomeCarousel>
      </div>
    </ListState>
  </div>
</template>
