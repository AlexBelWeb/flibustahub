<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Ellipsis } from '@lucide/vue'
import { Button, buttonVariants } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'
import { errorMessage } from '@/i18n/errors'
import { parseBackendError, type BackendError } from '@/lib/backend-error'
import { useStorageStore } from '@/stores/storage'
import { useToastStore } from '@/stores/toast'
import {
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogOverlay,
  AlertDialogPortal,
  AlertDialogRoot,
  AlertDialogTitle,
} from 'reka-ui'

const props = defineProps<{
  editionId?: number
  hasFile?: boolean
  quiet?: boolean
  hideDownload?: boolean
}>()

const { t } = useI18n()
const storage = useStorageStore()
const toast = useToastStore()
const busy = ref<'download' | 'read' | ''>('')
const slow = ref(false)
const lastPath = ref('')
const readerErr = ref<BackendError | null>(null)
let slowTimer: ReturnType<typeof setTimeout> | null = null
let canceling = false

const reason = computed(() => {
  if (!props.hasFile || !props.editionId) {
    return t('catalog.ghost')
  }
  if (!storage.configured) {
    return t('storage.noLibrary')
  }
  if (storage.unreachable) {
    return t('errors.library_unreachable')
  }
  if (!storage.available) {
    return t('errors.library_offline')
  }
  return ''
})

const blocked = computed(() => Boolean(reason.value) || Boolean(busy.value))

function clearSlow() {
  if (slowTimer) {
    clearTimeout(slowTimer)
    slowTimer = null
  }
  slow.value = false
}

function markBusy(kind: 'download' | 'read') {
  busy.value = kind
  clearSlow()
  slowTimer = setTimeout(() => {
    slow.value = true
  }, 1000)
}

async function ensureDownloadsDir(): Promise<boolean> {
  const path = await window.go.handlers.App.SelectDownloadsDir(t('settings.downloads.choose'))
  return Boolean(path)
}

async function run(kind: 'download' | 'read') {
  if (!props.editionId || reason.value || busy.value) {
    return
  }
  canceling = false
  markBusy(kind)
  try {
    const fn =
      kind === 'download'
        ? window.go.handlers.App.DownloadEdition
        : window.go.handlers.App.ReadEdition
    const out = await fn(props.editionId)
    lastPath.value = out.path
    if (kind === 'download') {
      toast.pushSuccess(t('book.downloaded', { name: out.fileName }))
    }
  } catch (err) {
    const be = parseBackendError(err)
    if (canceling || be.code === 'cancelled') {
      return
    }
    if (be.code === 'downloads_dir_unusable' && kind === 'download') {
      if (await ensureDownloadsDir()) {
        clearSlow()
        busy.value = ''
        await run(kind)
        return
      }
    }
    if (be.code === 'reader_unavailable' || be.code === 'reader_missing') {
      readerErr.value = be
      if (be.params?.file) {
        lastPath.value = be.params.file
      }
      return
    }
    toast.pushError(errorMessage(be.code, be.params))
  } finally {
    clearSlow()
    busy.value = ''
  }
}

function cancel() {
  if (!props.editionId || !busy.value) {
    return
  }
  canceling = true
  window.go.handlers.App.CancelFileOp(props.editionId, busy.value)
}

async function showFolder() {
  if (!lastPath.value) {
    return
  }
  try {
    await window.go.handlers.App.ShowInFolder(lastPath.value)
  } catch (err) {
    const be = parseBackendError(err)
    toast.pushError(errorMessage(be.code, be.params))
  }
}

async function chooseReader() {
  const path = await window.go.handlers.App.SelectReaderPath(t('settings.reader.choose'))
  if (path) {
    storage.readerPath = path
    readerErr.value = null
    await run('read')
  }
}

async function clearReader() {
  await window.go.handlers.App.ClearReaderPath()
  storage.readerPath = ''
  readerErr.value = null
  await run('read')
}
</script>

<template>
  <div
    :class="
      quiet
        ? 'flex flex-nowrap items-center justify-end gap-1'
        : 'flex flex-wrap items-center gap-2'
    "
  >
    <Tooltip :disabled="!reason">
      <TooltipTrigger as-child>
        <span class="inline-flex">
          <Button
            :variant="quiet ? 'ghost' : hideDownload ? 'outline' : 'default'"
            :size="quiet ? 'sm' : 'default'"
            :disabled="blocked && busy !== 'read'"
            @click="run('read')"
          >
            {{ busy === 'read' && slow ? t('book.readingDisk') : t('book.read') }}
          </Button>
        </span>
      </TooltipTrigger>
      <TooltipContent v-if="reason">{{ reason }}</TooltipContent>
    </Tooltip>
    <Tooltip v-if="!hideDownload" :disabled="!reason">
      <TooltipTrigger as-child>
        <span class="inline-flex">
          <Button
            :variant="quiet ? 'ghost' : 'outline'"
            :size="quiet ? 'sm' : 'default'"
            :disabled="blocked && busy !== 'download'"
            @click="run('download')"
          >
            {{ busy === 'download' && slow ? t('book.readingDisk') : t('book.download') }}
          </Button>
        </span>
      </TooltipTrigger>
      <TooltipContent v-if="reason">{{ reason }}</TooltipContent>
    </Tooltip>
    <Button v-if="busy && slow" variant="ghost" size="sm" @click="cancel">
      {{ t('common.cancel') }}
    </Button>
    <DropdownMenu v-if="lastPath && !busy && !hideDownload" :modal="false">
      <DropdownMenuTrigger
        :class="cn(buttonVariants({ variant: 'ghost', size: 'icon' }))"
        :aria-label="t('book.editionActions')"
      >
        <Ellipsis class="size-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DropdownMenuItem @click="showFolder">
          {{ t('book.showInFolder') }}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  </div>

  <AlertDialogRoot :open="!!readerErr" @update:open="(open) => !open && (readerErr = null)">
    <AlertDialogPortal>
      <AlertDialogOverlay class="fixed inset-0 z-[90] bg-scrim" />
      <AlertDialogContent
        class="fixed top-1/2 left-1/2 z-[90] w-[min(28rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-2xl dialog-surface p-6"
      >
        <AlertDialogTitle class="font-display text-lg">{{
          readerErr?.code === 'reader_missing'
            ? t('book.readerMissingTitle')
            : t('book.readerTitle')
        }}</AlertDialogTitle>
        <AlertDialogDescription class="mt-2 text-sm text-muted-foreground">
          {{
            readerErr
              ? errorMessage(readerErr.code, readerErr.params)
              : t('errors.reader_unavailable')
          }}
        </AlertDialogDescription>
        <div class="mt-6 flex flex-wrap justify-end gap-2">
          <AlertDialogCancel as-child>
            <Button variant="outline">{{ t('common.dismiss') }}</Button>
          </AlertDialogCancel>
          <Button v-if="lastPath" variant="outline" @click="showFolder">{{
            t('book.showInFolder')
          }}</Button>
          <AlertDialogAction v-if="readerErr?.code === 'reader_missing'" as-child>
            <Button variant="outline" @click="clearReader">{{ t('book.readerUseOs') }}</Button>
          </AlertDialogAction>
          <AlertDialogAction as-child>
            <Button @click="chooseReader">{{
              readerErr?.code === 'reader_missing' ? t('book.readerFix') : t('book.readerChoose')
            }}</Button>
          </AlertDialogAction>
        </div>
      </AlertDialogContent>
    </AlertDialogPortal>
  </AlertDialogRoot>
</template>
