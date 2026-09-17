<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { locales, type LocaleCode } from '@/i18n/registry'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const app = useAppStore()
</script>

<template>
  <div class="grid gap-6">
    <section class="grid gap-2">
      <h2 class="text-sm font-medium text-muted-foreground">
        {{ t('settings.interface.language') }}
      </h2>
      <div class="flex flex-wrap gap-2">
        <Button
          v-for="item in locales"
          :key="item.code"
          :variant="app.locale === item.code ? 'default' : 'outline'"
          @click="app.changeLocale(item.code as LocaleCode)"
        >
          {{ item.nativeName }}
        </Button>
      </div>
    </section>
    <section class="grid gap-2">
      <h2 class="text-sm font-medium text-muted-foreground">{{ t('settings.interface.theme') }}</h2>
      <div class="flex flex-wrap gap-2">
        <Button
          :variant="app.theme === 'system' ? 'default' : 'outline'"
          @click="app.changeTheme('system')"
        >
          {{ t('settings.interface.themeSystem') }}
        </Button>
        <Button
          :variant="app.theme === 'dark' ? 'default' : 'outline'"
          @click="app.changeTheme('dark')"
        >
          {{ t('settings.interface.themeDark') }}
        </Button>
        <Button
          :variant="app.theme === 'light' ? 'default' : 'outline'"
          @click="app.changeTheme('light')"
        >
          {{ t('settings.interface.themeLight') }}
        </Button>
      </div>
    </section>
    <section class="grid gap-2">
      <h2 class="text-sm font-medium text-muted-foreground">
        {{ t('settings.interface.effects') }}
      </h2>
      <div class="flex flex-wrap gap-2">
        <Button
          :variant="app.bootstrap?.visualEffectsPref === 'auto' ? 'default' : 'outline'"
          @click="app.changeEffects('auto')"
        >
          {{ t('settings.interface.effectsAuto') }}
        </Button>
        <Button
          :variant="app.bootstrap?.visualEffectsPref === 'full' ? 'default' : 'outline'"
          @click="app.changeEffects('full')"
        >
          {{ t('settings.interface.effectsFull') }}
        </Button>
        <Button
          :variant="app.bootstrap?.visualEffectsPref === 'reduced' ? 'default' : 'outline'"
          @click="app.changeEffects('reduced')"
        >
          {{ t('settings.interface.effectsReduced') }}
        </Button>
      </div>
      <p class="text-sm text-muted-foreground">{{ t('settings.interface.effectsHint') }}</p>
    </section>
  </div>
</template>
