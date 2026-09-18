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

export function formatBytes(value: number, locale: string): string {
  const units: Array<{ unit: Intl.NumberFormatOptions['unit']; size: number }> = [
    { unit: 'gigabyte', size: 1024 * 1024 * 1024 },
    { unit: 'megabyte', size: 1024 * 1024 },
    { unit: 'kilobyte', size: 1024 },
    { unit: 'byte', size: 1 },
  ]
  const found = units.find((item) => value >= item.size) ?? units[units.length - 1]
  const amount = value / found.size
  return new Intl.NumberFormat(locale, {
    style: 'unit',
    unit: found.unit,
    unitDisplay: 'short',
    maximumFractionDigits: amount >= 10 ? 0 : 1,
  }).format(amount)
}

export function formatDate(iso: string, locale: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) {
    return iso
  }
  return new Intl.DateTimeFormat(locale, { dateStyle: 'medium' }).format(date)
}

export function formatRelative(iso: string, locale: string, now = Date.now()): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) {
    return iso
  }
  const diffSec = Math.round((date.getTime() - now) / 1000)
  const abs = Math.abs(diffSec)
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' })
  if (abs < 60) {
    return rtf.format(diffSec, 'second')
  }
  if (abs < 3600) {
    return rtf.format(Math.round(diffSec / 60), 'minute')
  }
  if (abs < 86400) {
    return rtf.format(Math.round(diffSec / 3600), 'hour')
  }
  if (abs < 86400 * 30) {
    return rtf.format(Math.round(diffSec / 86400), 'day')
  }
  if (abs < 86400 * 365) {
    return rtf.format(Math.round(diffSec / (86400 * 30)), 'month')
  }
  return rtf.format(Math.round(diffSec / (86400 * 365)), 'year')
}
