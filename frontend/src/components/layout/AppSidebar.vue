<script setup lang="ts">
import { BookOpen, House, Layers, LayoutGrid, Settings, Sparkles, Users } from '@lucide/vue'
import { computed, type Component } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarTrigger,
} from '@/components/ui/sidebar'
import { useAppStore } from '@/stores/app'
import { useImportStore } from '@/stores/import'

const { t } = useI18n()
const app = useAppStore()
const imp = useImportStore()
const route = useRoute()

const items = computed((): Array<{ to: string; name: string; label: string; icon: Component }> => [
  { to: '/', name: 'home', label: t('nav.home'), icon: House },
  { to: '/books', name: 'books', label: t('nav.books'), icon: BookOpen },
  { to: '/authors', name: 'authors', label: t('nav.authors'), icon: Users },
  { to: '/series', name: 'series', label: t('nav.series'), icon: Layers },
  { to: '/genres', name: 'genres', label: t('nav.genres'), icon: LayoutGrid },
  {
    to: '/recommendations',
    name: 'recommendations',
    label: t('nav.recommendations'),
    icon: Sparkles,
  },
  { to: '/settings/interface', name: 'settings', label: t('nav.settings'), icon: Settings },
])

function active(name: string): boolean {
  if (name === 'home') {
    return route.path === '/'
  }
  if (name === 'settings') {
    return route.path.startsWith('/settings')
  }
  return route.path === `/${name}` || route.path.startsWith(`/${name}/`)
}

const activity = computed(() => {
  if (imp.running) {
    return t('activity.import')
  }
  if (!app.searchIndexReady) {
    return t('activity.index')
  }
  return ''
})
</script>

<template>
  <Sidebar collapsible="icon">
    <SidebarHeader>
      <SidebarTrigger :aria-label="app.sidebarIconsOnly ? t('nav.expand') : t('nav.collapse')" />
    </SidebarHeader>
    <SidebarContent>
      <SidebarGroup>
        <SidebarGroupContent>
          <SidebarMenu>
            <SidebarMenuItem v-for="item in items" :key="item.name">
              <SidebarMenuButton as-child :is-active="active(item.name)" :tooltip="item.label">
                <RouterLink :to="item.to" :aria-current="active(item.name) ? 'page' : undefined">
                  <component :is="item.icon" class="size-4" />
                  <span>{{ item.label }}</span>
                </RouterLink>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>
    </SidebarContent>
    <SidebarFooter v-if="activity">
      <p class="truncate px-2 text-xs text-muted-foreground">
        {{ activity }}
      </p>
    </SidebarFooter>
  </Sidebar>
</template>
