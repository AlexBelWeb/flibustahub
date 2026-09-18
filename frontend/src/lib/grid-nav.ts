export function nextGridIndex(
  current: number,
  key: string,
  count: number,
  columns: number,
): number | null {
  if (count <= 0) {
    return null
  }
  const cols = Math.max(1, columns)
  const from = current < 0 || current >= count ? 0 : current
  let next = from
  if (key === 'ArrowRight') {
    next = from + 1
  } else if (key === 'ArrowLeft') {
    next = from - 1
  } else if (key === 'ArrowDown') {
    next = from + cols
  } else if (key === 'ArrowUp') {
    next = from - cols
  } else if (key === 'Home') {
    next = 0
  } else if (key === 'End') {
    next = count - 1
  } else {
    return null
  }
  if (next < 0 || next >= count) {
    return from
  }
  return next
}

export function gridKeyHandled(key: string): boolean {
  return (
    key === 'ArrowRight' ||
    key === 'ArrowLeft' ||
    key === 'ArrowDown' ||
    key === 'ArrowUp' ||
    key === 'Home' ||
    key === 'End' ||
    key === 'Enter'
  )
}
