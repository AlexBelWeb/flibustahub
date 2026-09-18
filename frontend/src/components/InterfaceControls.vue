<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { locales, type LocaleCode } from '@/i18n/registry'
import { useAppStore } from '@/stores/app'
import type { Effects, Theme } from '@/stores/app'

const { t } = useI18n()
const app = useAppStore()
</script>

<template>
  <div class="grid gap-6">
    <section class="grid gap-2">
      <h2 class="text-sm font-medium text-muted-foreground">
        {{ t('settings.interface.language') }}
      </h2>
      <ToggleGroup
        type="single"
        :model-value="app.locale"
        @update:model-value="(value) => value && app.changeLocale(value as LocaleCode)"
      >
        <ToggleGroupItem v-for="item in locales" :key="item.code" :value="item.code">
          {{ item.nativeName }}
        </ToggleGroupItem>
      </ToggleGroup>
    </section>
    <section class="grid gap-2">
      <h2 class="text-sm font-medium text-muted-foreground">{{ t('settings.interface.theme') }}</h2>
      <ToggleGroup
        type="single"
        :model-value="app.theme"
        @update:model-value="(value) => value && app.changeTheme(value as Theme)"
      >
        <ToggleGroupItem value="system">{{ t('settings.interface.themeSystem') }}</ToggleGroupItem>
        <ToggleGroupItem value="dark">{{ t('settings.interface.themeDark') }}</ToggleGroupItem>
        <ToggleGroupItem value="light">{{ t('settings.interface.themeLight') }}</ToggleGroupItem>
      </ToggleGroup>
    </section>
    <section class="grid gap-2">
      <h2 class="text-sm font-medium text-muted-foreground">
        {{ t('settings.interface.effects') }}
      </h2>
      <ToggleGroup
        type="single"
        :model-value="app.bootstrap?.visualEffectsPref ?? 'auto'"
        @update:model-value="(value) => value && app.changeEffects(value as Effects)"
      >
        <ToggleGroupItem value="auto">{{ t('settings.interface.effectsAuto') }}</ToggleGroupItem>
        <ToggleGroupItem value="full">{{ t('settings.interface.effectsFull') }}</ToggleGroupItem>
        <ToggleGroupItem value="reduced">{{
          t('settings.interface.effectsReduced')
        }}</ToggleGroupItem>
      </ToggleGroup>
      <p class="text-sm text-muted-foreground">{{ t('settings.interface.effectsHint') }}</p>
    </section>
  </div>
</template>
