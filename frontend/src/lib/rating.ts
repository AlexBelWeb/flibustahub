export function starsFromInternal(rating: number | undefined | null): number {
  if (rating == null || rating < 1) {
    return 0
  }
  return rating / 2
}

export function internalFromStars(stars: number): number {
  if (!Number.isFinite(stars) || stars <= 0) {
    return 0
  }
  return Math.round(stars * 2)
}

export function formatStars(rating: number | undefined | null, locale: string): string {
  const stars = starsFromInternal(rating)
  if (!stars) {
    return ''
  }
  return new Intl.NumberFormat(locale, {
    minimumFractionDigits: stars % 1 === 0 ? 0 : 1,
    maximumFractionDigits: 1,
  }).format(stars)
}

export function ratingFromShortcut(event: KeyboardEvent): number | null {
  if (!event.code.startsWith('Digit')) {
    return null
  }
  const n = Number.parseInt(event.code.slice(5), 10)
  if (n < 1 || n > 5) {
    return null
  }
  return event.shiftKey ? n * 2 - 1 : n * 2
}
