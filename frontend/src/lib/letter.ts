export function letterLabel(key: string, yo: string, other: string): string {
  if (key === 'other') {
    return other
  }
  if (key === 'е') {
    return yo
  }
  return key.toLocaleUpperCase()
}
