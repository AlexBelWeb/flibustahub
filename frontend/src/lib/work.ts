const COVER_HUES = [12, 28, 160, 200, 255, 320, 340, 48]

// "0" is the dump's placeholder for a missing series number, not volume zero.
export function visibleSeriesNo(seriesNo: string | undefined): string {
  if (!seriesNo || seriesNo === '0') {
    return ''
  }
  return seriesNo
}

export function isBlankTitle(title: string | undefined): boolean {
  if (!title) {
    return true
  }
  return title.replace(/[\p{P}\p{S}\s]/gu, '') === ''
}

export function isUnknownAuthor(name: string | undefined): boolean {
  if (!name) {
    return true
  }
  const n = name.replace(/,/g, ' ').replace(/\s+/g, ' ').trim().toLowerCase()
  return n === '' || n === 'неизвестен автор'
}

export function authorParts(authorsText: string | undefined): string[] {
  if (!authorsText) {
    return []
  }
  return authorsText
    .split(',')
    .map((part) => part.trim())
    .filter(Boolean)
}

export function titleMonogram(title: string): string {
  const words = title
    .replace(/[^\p{L}\p{N}]+/gu, ' ')
    .trim()
    .split(/\s+/)
    .filter(Boolean)
  if (!words.length) {
    return '?'
  }
  const first = Array.from(words[0])
  if (words.length === 1) {
    return first.slice(0, 2).join('').toLocaleUpperCase()
  }
  const second = Array.from(words[1])
  return `${first[0] ?? ''}${second[0] ?? ''}`.toLocaleUpperCase()
}

export function coverHue(workKey: string): number {
  let hash = 0
  for (let i = 0; i < workKey.length; i += 1) {
    hash = (hash * 33 + workKey.charCodeAt(i)) >>> 0
  }
  return COVER_HUES[hash % COVER_HUES.length]
}

export function coverWash(workKey: string): string {
  const hue = coverHue(workKey)
  return `linear-gradient(105deg, hsl(${hue} 32% var(--cover-l, 42%) / 0.28) 0%, transparent 62%)`
}

export function coverTint(workKey: string): string {
  const hue = coverHue(workKey)
  return `hsl(${hue} 32% var(--cover-l, 42%) / 0.16)`
}
