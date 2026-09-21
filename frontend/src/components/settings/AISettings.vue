<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { errorMessage } from '@/i18n/errors'
import { useAppStore } from '@/stores/app'
import { useSecretsStore } from '@/stores/secrets'
import { useToastStore } from '@/stores/toast'
import { parseBackendError } from '@/lib/backend-error'
import { AI_PROVIDERS, type AIProviderId } from '@/types/secrets'

const NONE = 'none'

const { t } = useI18n()
const app = useAppStore()
const secrets = useSecretsStore()
const toast = useToastStore()

const provider = ref(NONE)
const secret = ref('')

const providerId = computed(() => (provider.value === NONE ? '' : provider.value))

onMounted(() => {
  const current = app.bootstrap?.aiProvider ?? ''
  provider.value = AI_PROVIDERS.includes(current as AIProviderId) ? current : NONE
  void secrets.load(providerId.value)
})

async function onProvider(value: unknown) {
  const next = String(value || NONE)
  if (next === provider.value) {
    return
  }
  const previous = provider.value
  provider.value = next
  secret.value = ''
  const id = next === NONE ? '' : next
  try {
    await secrets.setProvider(id)
  } catch (err) {
    const be = parseBackendError(err)
    toast.pushError(errorMessage(be.code, be.params))
    provider.value = previous
    return
  }
  await secrets.load(id)
}

async function save() {
  const id = providerId.value
  if (!id) {
    return
  }
  try {
    await secrets.saveSecret(id as AIProviderId, secret.value)
    secret.value = ''
    toast.pushInfo(t('settings.ai.saved'))
  } catch (err) {
    const be = parseBackendError(err)
    toast.pushError(errorMessage(be.code, be.params))
  }
}

async function remove() {
  const id = providerId.value
  if (!id) {
    return
  }
  try {
    await secrets.deleteSecret(id)
    secret.value = ''
    toast.pushInfo(t('settings.ai.deleted'))
  } catch (err) {
    const be = parseBackendError(err)
    toast.pushError(errorMessage(be.code, be.params))
  }
}

const storeLine = computed(() => {
  const st = secrets.status
  if (!st) {
    return ''
  }
  if (st.kind === 'file' && st.machineIdMissing) {
    return t('settings.ai.storeMachineId')
  }
  if (st.kind === 'os') {
    return t('settings.ai.storeOS')
  }
  return t('settings.ai.storeFile')
})

const unreadable = computed(() => secrets.loadError?.code === 'secret_store_unreadable')
const canDelete = computed(
  () => Boolean(providerId.value) && (Boolean(secrets.status?.hasSecret) || unreadable.value),
)
const canSave = computed(
  () => Boolean(providerId.value) && secret.value.length > 0 && !secrets.saving,
)
</script>

<template>
  <section class="rounded-2xl border border-border bg-card/80 p-6 backdrop-panel">
    <h2 class="font-display text-xl font-medium">{{ t('settings.ai.title') }}</h2>
    <p class="mt-2 text-sm text-muted-foreground">{{ t('settings.ai.lead') }}</p>

    <div
      v-if="secrets.loadState === 'loading' || secrets.loadState === 'idle'"
      class="mt-6 grid gap-3"
      aria-busy="true"
    >
      <span class="sr-only">{{ t('common.loading') }}</span>
      <Skeleton class="h-9 w-64" />
      <Skeleton class="h-9 w-full" />
      <Skeleton class="h-4 w-80" />
    </div>

    <div v-else-if="secrets.loadState === 'error'" class="mt-6 grid gap-3">
      <p>{{ secrets.errorText(secrets.loadError) }}</p>
      <div class="flex flex-wrap gap-2">
        <Button @click="secrets.load(providerId)">{{ t('common.retry') }}</Button>
        <Button
          v-if="unreadable && providerId"
          variant="outline"
          :disabled="secrets.saving"
          @click="remove"
        >
          {{ t('settings.ai.delete') }}
        </Button>
      </div>
    </div>

    <div v-else-if="secrets.loadState === 'empty'" class="mt-6">
      <p>{{ t('settings.ai.statusEmpty') }}</p>
      <Button class="mt-3" @click="secrets.load(providerId)">{{ t('common.retry') }}</Button>
    </div>

    <div v-else class="mt-6 grid gap-5">
      <div class="grid max-w-sm gap-2">
        <h3 class="text-sm font-medium text-muted-foreground">{{ t('settings.ai.provider') }}</h3>
        <Select :model-value="provider" @update:model-value="onProvider">
          <SelectTrigger :aria-label="t('settings.ai.provider')">
            <SelectValue :placeholder="t('settings.ai.providerNone')" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem :value="NONE">{{ t('settings.ai.providerNone') }}</SelectItem>
            <SelectItem value="gemini">{{ t('settings.ai.gemini') }}</SelectItem>
            <SelectItem value="openai">{{ t('settings.ai.openai') }}</SelectItem>
            <SelectItem value="ollama">{{ t('settings.ai.ollama') }}</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <p class="text-sm text-muted-foreground">{{ storeLine }}</p>
      <p v-if="secrets.status?.hasSecret" class="text-sm">{{ t('settings.ai.hasKey') }}</p>
      <p v-else class="text-sm text-muted-foreground">{{ t('settings.ai.noKey') }}</p>

      <div class="grid max-w-lg gap-2">
        <label class="text-sm font-medium text-muted-foreground" for="ai-secret">{{
          t('settings.ai.secret')
        }}</label>
        <Input
          id="ai-secret"
          v-model="secret"
          type="password"
          autocomplete="off"
          spellcheck="false"
          :disabled="!providerId || secrets.saving"
          :placeholder="t('settings.ai.secretPlaceholder')"
        />
      </div>

      <div class="flex flex-wrap gap-2">
        <Button :disabled="!canSave" @click="save">{{ t('settings.ai.save') }}</Button>
        <Button v-if="canDelete" variant="outline" :disabled="secrets.saving" @click="remove">
          {{ t('settings.ai.delete') }}
        </Button>
      </div>
      <p v-if="secrets.saveError" class="text-sm">{{ secrets.errorText(secrets.saveError) }}</p>
    </div>
  </section>
</template>
