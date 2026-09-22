<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import ImportHomeCard from '@/components/import/ImportHomeCard.vue'
import CoversSettings from '@/components/catalog/CoversSettings.vue'
import LibrarySettings from '@/components/catalog/LibrarySettings.vue'
import PersonalSettings from '@/components/catalog/PersonalSettings.vue'
import InterfaceControls from '@/components/InterfaceControls.vue'
import AISettings from '@/components/settings/AISettings.vue'
import DatabaseMaintenance from '@/components/settings/DatabaseMaintenance.vue'
import DiagnosticsSettings from '@/components/settings/DiagnosticsSettings.vue'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { useAppStore } from '@/stores/app'

const SETTINGS_SECTIONS = [
  'interface',
  'library',
  'downloads',
  'personal',
  'ai',
  'diagnostics',
] as const

type SettingsSection = (typeof SETTINGS_SECTIONS)[number]

function isSettingsSection(value: string): value is SettingsSection {
  return (SETTINGS_SECTIONS as readonly string[]).includes(value)
}

const { t } = useI18n()
const app = useAppStore()
const route = useRoute()
const router = useRouter()

const section = computed(() => {
  const raw = String(route.params.section || '')
  return isSettingsSection(raw) ? raw : 'interface'
})

const navItems = computed(() => [
  { id: 'interface' as const, label: t('settings.interface.title') },
  { id: 'library' as const, label: t('settings.library.title') },
  { id: 'downloads' as const, label: t('settings.downloads.navTitle') },
  { id: 'personal' as const, label: t('settings.personal.title') },
  { id: 'ai' as const, label: t('settings.ai.title') },
  { id: 'diagnostics' as const, label: t('settings.diagnostics.title') },
])

const sectionTitle = computed(() => {
  const item = navItems.value.find((entry) => entry.id === section.value)
  return item?.label ?? t('settings.title')
})

watch(
  () => String(route.params.section || ''),
  (value) => {
    if (!isSettingsSection(value)) {
      void router.replace({ name: 'settings', params: { section: 'interface' } })
    }
  },
  { immediate: true },
)

function go(value: unknown) {
  const id = String(value || '')
  if (isSettingsSection(id) && id !== section.value) {
    void router.push({ name: 'settings', params: { section: id } })
  }
}
</script>

<template>
  <div
    class="mx-auto flex min-h-0 w-full max-w-6xl flex-1 flex-col gap-6 overflow-hidden px-6 pt-10 pb-12"
  >
    <header class="shrink-0">
      <p class="text-sm text-muted-foreground">{{ t('settings.title') }}</p>
      <h1 class="font-display text-3xl font-semibold">{{ sectionTitle }}</h1>
      <div v-if="app.narrow" class="mt-4 max-w-md">
        <Select :model-value="section" @update:model-value="go">
          <SelectTrigger :aria-label="t('settings.navLabel')">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="item in navItems" :key="item.id" :value="item.id">
              {{ item.label }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
    </header>

    <div class="flex min-h-0 flex-1 gap-8 overflow-hidden">
      <nav
        v-if="!app.narrow"
        class="w-56 shrink-0 overflow-auto pb-8"
        :aria-label="t('settings.navLabel')"
      >
        <ul class="grid gap-1">
          <li v-for="item in navItems" :key="item.id">
            <Button
              variant="ghost"
              class="h-auto w-full justify-start px-3 py-2 text-left whitespace-normal"
              :class="
                item.id === section ? 'bg-accent text-accent-foreground' : 'text-muted-foreground'
              "
              @click="go(item.id)"
            >
              {{ item.label }}
            </Button>
          </li>
        </ul>
      </nav>

      <div class="scroll-stable min-h-0 min-w-0 flex-1 overflow-auto pb-8">
        <Skeleton v-if="app.loading" class="h-48 rounded-2xl" />
        <Card v-else-if="app.loadError" class="p-6">
          <p class="mb-4">{{ t('errors.internal') }}</p>
          <Button @click="app.load()">{{ t('common.retry') }}</Button>
        </Card>
        <section v-else-if="section === 'library'" class="grid gap-0">
          <ImportHomeCard />
          <DatabaseMaintenance />
          <CoversSettings />
        </section>
        <section v-else-if="section === 'downloads'">
          <LibrarySettings />
        </section>
        <section v-else-if="section === 'personal'">
          <PersonalSettings />
        </section>
        <section v-else-if="section === 'ai'">
          <AISettings />
        </section>
        <section v-else-if="section === 'diagnostics'">
          <DiagnosticsSettings />
        </section>
        <Card v-else class="bg-card/80 p-6 backdrop-panel">
          <InterfaceControls />
        </Card>
      </div>
    </div>
  </div>
</template>
