export function queryText(value: unknown): string {
  if (Array.isArray(value)) {
    return queryText(value[0])
  }
  return typeof value === 'string' ? value : ''
}

export function queryId(value: unknown): number {
  const raw = queryText(value)
  if (!raw) {
    return 0
  }
  const n = Number.parseInt(raw, 10)
  return Number.isFinite(n) && n > 0 ? n : 0
}

export function queryFlag(value: unknown): boolean {
  const raw = queryText(value)
  return raw === '1' || raw.toLowerCase() === 'true'
}

export function routeId(value: string | string[] | undefined): number {
  const raw = Array.isArray(value) ? value[0] : value
  return queryId(raw)
}
