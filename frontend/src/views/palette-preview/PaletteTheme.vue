<script setup lang="ts">
import { Check, Library, Star, Trash2, TriangleAlert } from '@lucide/vue'
import type { Component } from 'vue'
import { Skeleton } from '@/components/ui/skeleton'
import ContrastMark from '@/views/palette-preview/ContrastMark.vue'

const props = defineProps<{
  theme: 'light' | 'dark'
  neutral: 'charcoal' | 'warm'
  tick: number
}>()

const surfaces = [
  { name: 'Фон', bg: '--background', lift: 'none' },
  { name: 'Карточка', bg: '--card', lift: 'card' },
  { name: 'Поповер', bg: '--popover', lift: 'pop' },
  { name: 'Сайдбар', bg: '--sidebar', lift: 'side' },
] as const

const roles: Array<{
  id: string
  title: string
  word: string
  icon: Component
}> = [
  { id: 'brand', title: 'Бренд', word: 'Оценка', icon: Star },
  { id: 'danger', title: 'Опасность', word: 'Удалить', icon: Trash2 },
  { id: 'success', title: 'Успех', word: 'Готово', icon: Check },
  { id: 'warning', title: 'Предупреждение', word: 'Нет диска', icon: TriangleAlert },
  { id: 'library', title: 'Данные библиотеки', word: 'В дампе', icon: Library },
]

const covers = [0, 30, 60, 90, 120, 150, 180, 210, 240, 270, 300, 330]
const monograms = ['АБ', 'ВГ', 'ДЕ', 'ЖЗ', 'ИК', 'ЛМ', 'НО', 'ПР', 'СТ', 'УФ', 'ХЦ', 'ЧШ']

function tone(id: string): Record<string, string> {
  return {
    '--tone': `var(--${id})`,
    '--tone-fg': `var(--${id}-foreground)`,
    '--tone-quiet': `var(--${id}-quiet)`,
    '--tone-quiet-fg': `var(--${id}-quiet-foreground)`,
    '--tone-l': `var(--${id}-l)`,
    '--tone-c': `var(--${id}-c)`,
    '--tone-h': `var(--${id}-h)`,
  }
}

function coverStyle(hue: number): Record<string, string> {
  return {
    background: `oklch(var(--cover-l) var(--cover-c) ${hue})`,
    color: 'var(--cover-fg)',
  }
}
</script>

<template>
  <section
    class="theme"
    :class="props.theme === 'light' ? 'palette-light' : 'palette-dark'"
    :data-neutral="props.neutral"
  >
    <h2>{{ props.theme === 'light' ? 'Светлая тема' : 'Тёмная тема' }}</h2>

    <div class="block">
      <h3>Поверхности</h3>
      <div class="surface-row">
        <div
          v-for="surface in surfaces"
          :key="surface.name"
          class="surface"
          :class="'lift-' + surface.lift"
          :style="{ background: `var(${surface.bg})`, color: 'var(--foreground)' }"
        >
          <strong>{{ surface.name }}</strong>
          <p class="quiet-text">Вторичный текст</p>
          <ContrastMark
            fg="var(--muted-foreground)"
            :bg="`var(${surface.bg})`"
            :tick="props.tick"
          />
        </div>
      </div>
      <div class="hairline">
        <span>Граница 1 px на карточке</span>
        <ContrastMark fg="var(--border)" bg="var(--card)" :tick="props.tick" />
      </div>
      <div class="on-muted">
        <span>Вторичный текст на muted</span>
        <ContrastMark fg="var(--muted-foreground)" bg="var(--muted)" :tick="props.tick" />
      </div>
    </div>

    <div class="block">
      <h3>Бренд и опасность рядом</h3>
      <div class="compare">
        <div
          class="swatch"
          :style="{ background: 'var(--brand)', color: 'var(--brand-foreground)' }"
        >
          <span class="huge">Бренд</span>
        </div>
        <div
          class="swatch"
          :style="{ background: 'var(--danger)', color: 'var(--danger-foreground)' }"
        >
          <span class="huge">Опасность</span>
        </div>
      </div>
      <div class="mark-row">
        <span>Текст на заливке бренда</span>
        <ContrastMark fg="var(--brand-foreground)" bg="var(--brand)" :tick="props.tick" />
        <span>Текст на заливке опасности</span>
        <ContrastMark fg="var(--danger-foreground)" bg="var(--danger)" :tick="props.tick" />
      </div>
      <div class="compare small">
        <span class="small-brand">бренд</span>
        <span class="small-danger">опасность</span>
        <span class="badge" :style="tone('brand')">личное</span>
        <span class="badge" :style="tone('danger')">удалить</span>
      </div>
      <div class="mark-row">
        <span>Мелкий бренд на фоне</span>
        <ContrastMark fg="var(--brand)" bg="var(--background)" :tick="props.tick" />
        <span>Мелкая опасность на фоне</span>
        <ContrastMark fg="var(--danger)" bg="var(--background)" :tick="props.tick" />
      </div>
      <div class="states">
        <button type="button" class="spec fill is-focus" :style="tone('brand')">
          Бренд в фокусе
        </button>
        <button type="button" class="spec fill is-focus" :style="tone('danger')">
          Удалить в фокусе
        </button>
      </div>
      <p class="caption">
        Кольцо фокуса — 2 px, отступ 2 px, цвет бренда. На опасной кнопке заливка и кольцо разных
        тонов.
      </p>
    </div>

    <div v-for="role in roles" :key="role.id" class="block role" :style="tone(role.id)">
      <h3>{{ role.title }}</h3>
      <div class="forms">
        <div class="chip on-bg">
          <span class="sample">Текст на фоне</span>
          <ContrastMark :fg="`var(--${role.id})`" bg="var(--background)" :tick="props.tick" />
        </div>
        <div class="chip on-card lift-card">
          <span class="sample">Текст на карточке</span>
          <ContrastMark :fg="`var(--${role.id})`" bg="var(--card)" :tick="props.tick" />
        </div>
        <button type="button" class="spec fill">Залитая</button>
        <ContrastMark
          :fg="`var(--${role.id}-foreground)`"
          :bg="`var(--${role.id})`"
          :tick="props.tick"
        />
        <button type="button" class="spec quiet">Тихая</button>
        <span class="badge">Бейдж</span>
        <ContrastMark
          :fg="`var(--${role.id}-quiet-foreground)`"
          :bg="`var(--${role.id}-quiet)`"
          :tick="props.tick"
        />
        <span class="icon-line">
          <component :is="role.icon" class="size-5" aria-hidden="true" />
          {{ role.word }}
        </span>
        <div class="bar" />
      </div>
      <div class="states">
        <button type="button" class="spec fill">Покой</button>
        <button type="button" class="spec fill is-hover">Наведение</button>
        <button type="button" class="spec fill is-active">Нажатие</button>
        <button type="button" class="spec fill" disabled>Задисейблено</button>
        <button type="button" class="spec fill is-focus">Фокус</button>
        <button type="button" class="spec fill live">Живая</button>
      </div>
    </div>

    <div class="block">
      <h3>Цвет книги</h3>
      <p class="caption">
        Светлота и насыщенность от темы, тон задан числом. Буквы — монограммный фоллбэк.
      </p>
      <div class="covers">
        <div v-for="(hue, index) in covers" :key="hue" class="cover" :style="coverStyle(hue)">
          <span class="mono">{{ monograms[index] }}</span>
          <span class="hue">{{ hue }}</span>
          <ContrastMark
            fg="var(--cover-fg)"
            :bg="`oklch(var(--cover-l) var(--cover-c) ${hue})`"
            :tick="props.tick"
          />
        </div>
      </div>
    </div>

    <div class="block">
      <h3>Диалог на скриме</h3>
      <div class="scrim">
        <div class="dialog lift-pop">
          <p class="dialog-title">Удалить ключ</p>
          <p class="quiet-text">Ключ будет удалён из хранилища.</p>
          <ContrastMark fg="var(--muted-foreground)" bg="var(--popover)" :tick="props.tick" />
          <div class="states">
            <button type="button" class="spec fill is-focus" :style="tone('danger')">
              Удалить
            </button>
            <button type="button" class="spec quiet" :style="tone('brand')">Оставить</button>
          </div>
        </div>
      </div>
    </div>

    <div class="block">
      <h3>Скелетон</h3>
      <div class="skel-row">
        <div class="skel-card lift-card">
          <Skeleton class="h-[6.75rem] w-[4.5rem]" />
          <Skeleton class="h-3 w-full" />
          <Skeleton class="h-3 w-3/5" />
        </div>
        <div class="skel-table lift-card">
          <Skeleton class="h-3 flex-[2]" />
          <Skeleton class="h-3 flex-1" />
          <Skeleton class="h-3 flex-1" />
          <Skeleton class="h-3 w-12" />
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.theme {
  display: grid;
  gap: 1.25rem;
  padding: 1.25rem;
  border: 1px solid var(--border);
  border-radius: 1rem;
  background: var(--background);
  color: var(--foreground);
}

h2,
h3 {
  margin: 0;
  font-weight: 600;
}

h2 {
  font-size: 1.25rem;
}

h3 {
  font-size: 0.95rem;
}

.block {
  display: grid;
  gap: 0.75rem;
}

.caption,
.quiet-text {
  margin: 0;
  color: var(--muted-foreground);
  font-size: 0.875rem;
}

.surface-row,
.compare,
.forms,
.states,
.covers,
.skel-row,
.mark-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: stretch;
}

.surface,
.chip,
.hairline,
.on-muted,
.dialog,
.skel-card,
.skel-table {
  border-radius: 0.75rem;
}

.surface,
.chip {
  display: grid;
  gap: 0.35rem;
  align-content: start;
  min-width: 9rem;
  padding: 0.75rem;
}

.on-bg {
  background: var(--background);
}

.on-card,
.hairline,
.on-muted,
.skel-card,
.skel-table {
  background: var(--card);
  color: var(--card-foreground);
}

.hairline {
  padding: 0.75rem;
  border: 1px solid var(--border);
}

.on-muted {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: baseline;
  padding: 0.75rem;
  background: var(--muted);
  color: var(--muted-foreground);
}

.lift-card {
  box-shadow: var(--palette-shadow);
}

.lift-pop {
  box-shadow: var(--palette-shadow-pop);
}

.lift-side {
  box-shadow: var(--palette-shadow-side);
}

.swatch {
  display: grid;
  place-items: center;
  width: 9rem;
  min-height: 6.5rem;
  border-radius: 0.75rem;
}

.huge {
  font-size: 2rem;
  font-weight: 650;
  line-height: 1;
}

.small-brand,
.small-danger {
  font-size: 0.75rem;
}

.small-brand {
  color: var(--brand);
}

.small-danger {
  color: var(--danger);
}

.sample,
.icon-line,
.bar {
  color: var(--tone);
}

.icon-line {
  display: inline-flex;
  gap: 0.35rem;
  align-items: center;
}

.bar {
  align-self: center;
  width: 6rem;
  height: 0.35rem;
  border-radius: 999px;
  background: var(--tone);
}

.spec {
  font: inherit;
  border: 0;
  border-radius: 0.5rem;
  padding: 0.4rem 0.75rem;
  cursor: pointer;
}

.spec:disabled {
  cursor: not-allowed;
  opacity: 0.4;
}

.fill {
  background: var(--tone);
  color: var(--tone-fg);
}

.quiet,
.badge {
  background: var(--tone-quiet);
  color: var(--tone-quiet-fg);
}

.badge {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  padding: 0.15rem 0.55rem;
  font-size: 0.75rem;
}

.palette-light .is-hover {
  background: oklch(calc(var(--tone-l) - 0.05) var(--tone-c) var(--tone-h));
}

.palette-light .is-active,
.palette-light .live:active {
  background: oklch(calc(var(--tone-l) - 0.09) var(--tone-c) var(--tone-h));
}

.palette-dark .is-hover,
.palette-dark .live:hover {
  background: oklch(calc(var(--tone-l) + 0.04) var(--tone-c) var(--tone-h));
}

.palette-dark .is-active,
.palette-dark .live:active {
  background: oklch(calc(var(--tone-l) + 0.08) var(--tone-c) var(--tone-h));
}

.palette-light .live:hover {
  background: oklch(calc(var(--tone-l) - 0.05) var(--tone-c) var(--tone-h));
}

.is-focus,
.live:focus-visible {
  outline: 2px solid var(--ring);
  outline-offset: 2px;
}

.covers {
  align-items: start;
}

.cover {
  display: grid;
  gap: 0.25rem;
  justify-items: center;
  width: 5.5rem;
  padding: 0.5rem 0.35rem 0.4rem;
  border-radius: 0.5rem;
}

.mono {
  font-size: 1.35rem;
  font-weight: 650;
  letter-spacing: 0.04em;
}

.hue {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  opacity: 0.85;
}

.scrim {
  display: grid;
  padding: 1.5rem;
  border-radius: 0.75rem;
  background: var(--palette-scrim);
  place-items: center;
}

.dialog {
  display: grid;
  gap: 0.6rem;
  width: min(100%, 22rem);
  padding: 1.1rem;
  background: var(--popover);
  color: var(--popover-foreground);
}

.dialog-title {
  margin: 0;
  font-size: 1.05rem;
  font-weight: 600;
}

.skel-card,
.skel-table {
  display: flex;
  gap: 0.6rem;
  padding: 0.75rem;
}

.skel-card {
  flex-direction: column;
  width: 8.5rem;
}

.skel-table {
  flex: 1;
  align-items: center;
  min-width: 16rem;
}
</style>
