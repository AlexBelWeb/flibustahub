<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import PaletteTheme from '@/views/palette-preview/PaletteTheme.vue'

// Temporary route. Removed before this slice closes. Not part of the shell.

type RoleId = 'brand' | 'danger' | 'success' | 'warning' | 'library'

const groups: Array<{
  id: RoleId
  label: string
  options: Array<{ h: number; label: string }>
}> = [
  {
    id: 'brand',
    label: 'Бренд',
    options: [
      { h: 31, label: 'Терракота' },
      { h: 46, label: 'Глина' },
    ],
  },
  {
    id: 'danger',
    label: 'Опасность',
    options: [
      { h: 15, label: 'Холодный багрянец' },
      { h: 40, label: 'Кирпичный' },
    ],
  },
  {
    id: 'success',
    label: 'Успех',
    options: [
      { h: 115, label: 'Олива' },
      { h: 168, label: 'Патина' },
    ],
  },
  {
    id: 'warning',
    label: 'Предупреждение',
    options: [
      { h: 84, label: 'Охра' },
      { h: 58, label: 'Янтарь' },
    ],
  },
  {
    id: 'library',
    label: 'Данные библиотеки',
    options: [
      { h: 230, label: 'Сталь' },
      { h: 258, label: 'Графитовый синий' },
    ],
  },
]

const selected = reactive<Record<RoleId, number>>({
  brand: 46,
  danger: 15,
  success: 168,
  warning: 58,
  library: 230,
})

const neutral = ref<'charcoal' | 'warm'>('warm')
const tick = ref(0)

const hueStyle = computed(() => ({
  '--brand-h': String(selected.brand),
  '--danger-h': String(selected.danger),
  '--success-h': String(selected.success),
  '--warning-h': String(selected.warning),
  '--library-h': String(selected.library),
}))

watch(
  () =>
    [
      selected.brand,
      selected.danger,
      selected.success,
      selected.warning,
      selected.library,
      neutral.value,
    ] as const,
  () => {
    tick.value += 1
  },
)
</script>

<template>
  <div class="preview">
    <header class="toolbar">
      <div class="intro">
        <h1>Палитра ролей</h1>
        <p>
          Временная страница, в меню её нет. Перед закрытием среза она удаляется. В приложении стоят
          выбранные тона; здесь их всё ещё можно сравнить.
        </p>
        <p>Рядом с парой — контраст. 4.5 — обычный текст, 3.0 — крупный текст и границы.</p>
      </div>
      <div class="groups">
        <fieldset v-for="group in groups" :key="group.id">
          <legend>{{ group.label }}</legend>
          <button
            v-for="option in group.options"
            :key="option.h"
            type="button"
            :aria-pressed="selected[group.id] === option.h"
            @click="selected[group.id] = option.h"
          >
            {{ option.label }}
            <span class="deg">{{ option.h }}°</span>
          </button>
        </fieldset>
        <fieldset>
          <legend>Нейтрали тёмной темы</legend>
          <button
            type="button"
            :aria-pressed="neutral === 'charcoal'"
            @click="neutral = 'charcoal'"
          >
            Нейтральный уголь
          </button>
          <button type="button" :aria-pressed="neutral === 'warm'" @click="neutral = 'warm'">
            Тёплый подтон
          </button>
        </fieldset>
      </div>
    </header>
    <div class="themes" :style="hueStyle">
      <PaletteTheme theme="light" :neutral="neutral" :tick="tick" />
      <PaletteTheme theme="dark" :neutral="neutral" :tick="tick" />
    </div>
  </div>
</template>

<style scoped>
.preview {
  height: 100%;
  overflow: auto;
  background: var(--background);
  color: var(--foreground);
}

.toolbar {
  position: sticky;
  top: 0;
  z-index: 2;
  display: grid;
  gap: 0.75rem;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--border);
  background: var(--background);
}

.intro h1 {
  margin: 0 0 0.35rem;
  font-size: 1.35rem;
  font-weight: 650;
}

.intro p {
  margin: 0;
  max-width: 46rem;
  color: var(--muted-foreground);
  font-size: 0.9rem;
}

.groups {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem 1rem;
}

fieldset {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  margin: 0;
  padding: 0.35rem 0.5rem 0.5rem;
  border: 1px solid var(--border);
  border-radius: 0.6rem;
}

legend {
  padding: 0 0.25rem;
  font-size: 0.75rem;
}

.toolbar button {
  font: inherit;
  border: 1px solid var(--border);
  border-radius: 999px;
  padding: 0.2rem 0.65rem;
  background: var(--card);
  color: var(--foreground);
  cursor: pointer;
}

.toolbar button[aria-pressed='true'] {
  background: var(--foreground);
  color: var(--background);
}

.deg {
  font-family: var(--font-mono);
  font-size: 0.75rem;
}

.themes {
  display: grid;
  gap: 1rem;
  padding: 1rem 1.25rem 2rem;
}

@media (min-width: 1200px) {
  .themes {
    grid-template-columns: 1fr 1fr;
  }
}
</style>
