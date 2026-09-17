export interface BackendError {
  code: string
  params: Record<string, string>
}

export function parseBackendError(err: unknown): BackendError {
  const raw = err instanceof Error ? err.message : String(err)
  try {
    const parsed = JSON.parse(raw) as { code?: string; params?: Record<string, string> }
    if (parsed && typeof parsed.code === 'string') {
      return { code: parsed.code, params: parsed.params ?? {} }
    }
  } catch {
    // Wails may wrap the payload; try to find a JSON object inside the string.
    const start = raw.indexOf('{')
    const end = raw.lastIndexOf('}')
    if (start >= 0 && end > start) {
      try {
        const parsed = JSON.parse(raw.slice(start, end + 1)) as {
          code?: string
          params?: Record<string, string>
        }
        if (parsed && typeof parsed.code === 'string') {
          return { code: parsed.code, params: parsed.params ?? {} }
        }
      } catch {
        // fall through
      }
    }
  }
  return { code: 'internal', params: {} }
}

export function errorI18nKey(code: string): string {
  return `errors.${code}`
}
