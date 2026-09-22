<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Progress } from '@/components/ui/progress'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

const { t } = useI18n()
</script>

<template>
  <div
    class="mx-auto flex min-h-screen max-w-xl flex-col justify-center gap-6 px-6"
    aria-busy="true"
  >
    <Card class="p-0">
      <CardHeader class="p-8 pb-0">
        <CardTitle class="font-display text-3xl">{{ t('startup.updatingTitle') }}</CardTitle>
      </CardHeader>
      <CardContent class="p-8 pt-3">
        <p class="text-muted-foreground">{{ t('startup.updatingLead') }}</p>
        <div class="mt-8">
          <Progress :model-value="null" :label="t('startup.updatingTitle')" />
        </div>
      </CardContent>
    </Card>
  </div>
</template>
