export interface PersonalSnapshot {
  unsyncedCount: number
  lastExportAt?: string
}

export interface PersonalExportResult {
  path: string
  count: number
}

export interface PersonalImportNote {
  workKey?: string
  title?: string
  field?: string
  reason: string
}

export interface PersonalImportReport {
  applied: number
  skipped: number
  notFound: number
  invalid: number
  notes?: PersonalImportNote[]
  notFoundPath?: string
}

export function asPersonalSnapshot(value: PersonalSnapshot | null | undefined): PersonalSnapshot {
  return {
    unsyncedCount: value?.unsyncedCount ?? 0,
    lastExportAt: value?.lastExportAt,
  }
}

export function asImportReport(
  value: PersonalImportReport | null | undefined,
): PersonalImportReport {
  return {
    applied: value?.applied ?? 0,
    skipped: value?.skipped ?? 0,
    notFound: value?.notFound ?? 0,
    invalid: value?.invalid ?? 0,
    notes: value?.notes ?? [],
    notFoundPath: value?.notFoundPath,
  }
}
