<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import ImportHomeCard from '@/components/import/ImportHomeCard.vue'
import CoversSettings from '@/components/catalog/CoversSettings.vue'
import LibrarySettings from '@/components/catalog/LibrarySettings.vue'
import InterfaceControls from '@/components/InterfaceControls.vue'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const app = useAppStore()
const route = useRoute()
const router = useRouter()
const section = computed(() => String(route.params.section || 'interface'))

function onSection(value: string | number) {
  void router.push({ name: 'settings', params: { section: String(value) } })
}
</script>

<template>
  <div class="mx-auto flex min-h-0 flex-1 flex-col gap-8 overflow-auto px-6 pt-10 pb-12">
    <header>
      <p class="text-sm text-muted-foreground">{{ t('settings.title') }}</p>
      <h1 class="font-display text-3xl font-semibold">
        {{ section === 'library' ? t('settings.library.title') : t('settings.interface.title') }}
      </h1>
      <Tabs class="mt-4" :model-value="section" @update:model-value="onSection">
        <TabsList>
          <TabsTrigger value="interface">{{ t('settings.interface.title') }}</TabsTrigger>
          <TabsTrigger value="library">{{ t('settings.library.title') }}</TabsTrigger>
        </TabsList>
      </Tabs>
    </header>

    <Skeleton v-if="app.loading" class="h-48 rounded-2xl" />
    <Card v-else-if="app.loadError" class="p-6">
      <p class="mb-4">{{ t('errors.internal') }}</p>
      <Button @click="app.load()">{{ t('common.retry') }}</Button>
    </Card>
    <section v-else-if="section === 'library'">
      <ImportHomeCard />
      <LibrarySettings />
      <CoversSettings />
    </section>
    <Card v-else class="bg-card/80 p-6 backdrop-panel">
      <InterfaceControls />
      <dl class="mt-8 grid gap-2 font-mono text-xs text-muted-foreground tabular-nums">
        <div class="flex gap-2">
          <dt>{{ t('status.version') }}</dt>
          <dd>{{ app.bootstrap?.version }}</dd>
        </div>
        <div class="flex gap-2">
          <dt>{{ t('status.platform') }}</dt>
          <dd>{{ app.bootstrap?.capabilities.os }}</dd>
        </div>
        <div class="flex gap-2">
          <dt>{{ t('status.effects') }}</dt>
          <dd>{{ app.reducedEffects ? t('status.effectsReduced') : t('status.effectsFull') }}</dd>
        </div>
      </dl>
    </Card>
  </div>
</template>
