<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { RouterView, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppSidebar from '@/components/layout/AppSidebar.vue'
import CommandPalette from '@/components/search/CommandPalette.vue'
import OnboardingWizard from '@/components/onboarding/OnboardingWizard.vue'
import { Button } from '@/components/ui/button'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
import { useAppStore } from '@/stores/app'
import { useImportStore } from '@/stores/import'
import { useUiStore } from '@/stores/ui'

const { t } = useI18n()
const app = useAppStore()
const imp = useImportStore()
const ui = useUiStore()
const router = useRouter()

const blocked = computed(() => imp.modalOpen || ui.paletteOpen || ui.onboardingOpen)
const sidebarOpen = computed(() => !app.sidebarCollapsed && !app.narrow)

function onSidebarOpen(open: boolean) {
  if (app.narrow) {
    void app.toggleSidebar()
    return
  }
  void app.setSidebarCollapsed(!open)
}

function onKey(event: KeyboardEvent) {
  const mod = event.ctrlKey || event.metaKey
  const target = event.target
  const typing =
    target instanceof HTMLInputElement ||
    target instanceof HTMLTextAreaElement ||
    (target instanceof HTMLElement && target.isContentEditable)

  if (mod && event.key.toLowerCase() === 'k') {
    event.preventDefault()
    if (!imp.modalOpen) {
      ui.togglePalette()
    }
    return
  }
  if (event.key === '/' && !typing && !blocked.value) {
    event.preventDefault()
    if (
      router.currentRoute.value.path.startsWith('/books') ||
      router.currentRoute.value.path.startsWith('/authors') ||
      router.currentRoute.value.path.startsWith('/series') ||
      router.currentRoute.value.path.startsWith('/genres')
    ) {
      ui.requestSearchFocus()
    } else {
      ui.openPalette()
    }
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKey)
})
onUnmounted(() => {
  window.removeEventListener('keydown', onKey)
})
</script>

<template>
  <SidebarProvider :open="sidebarOpen" @update:open="onSidebarOpen">
    <AppSidebar />
    <SidebarInset>
      <header class="flex items-center gap-3 border-b border-border px-4 py-2">
        <Button variant="outline" size="sm" @click="ui.openPalette()">
          {{ t('nav.search') }}
        </Button>
      </header>
      <div class="relative min-h-0 flex-1">
        <div class="absolute inset-0 flex min-h-0 flex-col overflow-hidden">
          <RouterView />
        </div>
      </div>
    </SidebarInset>
    <CommandPalette />
    <OnboardingWizard />
  </SidebarProvider>
</template>
