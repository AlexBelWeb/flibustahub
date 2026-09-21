<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue'
import { contrastOf } from '@/lib/contrast'

const props = defineProps<{
  fg: string
  bg: string
  tick: number
}>()

const probe = ref<HTMLElement | null>(null)
const ratio = ref<number | null>(null)

const textPass = (value: number | null) => value !== null && value >= 4.5
const largePass = (value: number | null) => value !== null && value >= 3

function measure() {
  const el = probe.value
  if (!el) {
    return
  }
  const style = getComputedStyle(el)
  ratio.value = contrastOf(style.color, style.backgroundColor)
}

onMounted(() => {
  void nextTick(measure)
})
watch(
  () => props.tick,
  () => {
    void nextTick(measure)
  },
)
</script>

<template>
  <span class="mark">
    <span ref="probe" class="probe" :style="{ color: fg, background: bg }" />
    <span class="num">{{ ratio === null ? '—' : ratio.toFixed(2) }}</span>
    <template v-if="ratio !== null">
      <span :data-pass="textPass(ratio)">4.5 {{ textPass(ratio) ? 'да' : 'нет' }}</span>
      <span :data-pass="largePass(ratio)">3.0 {{ largePass(ratio) ? 'да' : 'нет' }}</span>
    </template>
  </span>
</template>

<style scoped>
.mark {
  display: inline-flex;
  gap: 0.35rem;
  align-items: baseline;
  padding: 0.05rem 0.3rem;
  border-radius: 0.25rem;
  background: var(--background);
  font-family: var(--font-mono);
  font-size: 0.75rem;
  font-weight: 400;
  color: var(--muted-foreground);
}

.probe {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  pointer-events: none;
  clip-path: inset(50%);
}

.num {
  color: var(--foreground);
}

.mark [data-pass='false'] {
  font-weight: 700;
  text-decoration: underline;
  color: var(--foreground);
}
</style>
