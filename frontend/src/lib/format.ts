export function formatCount(value: number, locale: string): string {
  return new Intl.NumberFormat(locale, { useGrouping: true }).format(value)
}

export function formatDuration(ms: number, locale: string): string {
  if (!Number.isFinite(ms) || ms < 0) {
    return formatCount(0, locale)
  }
  if (ms < 1000) {
    return new Intl.NumberFormat(locale, { maximumFractionDigits: 0 }).format(ms)
  }
  const seconds = ms / 1000
  if (seconds < 10) {
    return new Intl.NumberFormat(locale, {
      minimumFractionDigits: 1,
      maximumFractionDigits: 1,
    }).format(seconds)
  }
  return new Intl.NumberFormat(locale, { maximumFractionDigits: 0 }).format(Math.round(seconds))
}

export function bytesPercent(done: number, total: number): number {
  if (total <= 0) {
    return 0
  }
  return Math.max(0, Math.min(100, (done / total) * 100))
}
