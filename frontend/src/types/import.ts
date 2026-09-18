export interface ImportProgress {
  phase: 'reading' | 'records' | 'fts' | 'warmup' | string
  recordsSeen: number
  bytesDone: number
  bytesTotal: number
  committed: boolean
}

export interface ImportEncodings {
  utf8: number
  cp1251: number
  versionInfo?: string
  collectionInfo?: string
}

export interface ImportNotes {
  missingArchives: string[]
  missingArchivesTotal: number
  unnamedGenres: string[]
  unnamedGenresTotal: number
  skippedMalformed: number
  skippedNoLibid: number
  encodings: ImportEncodings
  genreNamesMapped: number
  phasesMs?: Record<string, number>
}

export interface ImportReport {
  id: number
  status: string
  inpxPath: string
  inpxVersion: string
  finishedAt?: string
  recordsSeen: number
  worksAdded: number
  editionsAdded: number
  editionsUpdated: number
  editionsDeactivated: number
  libidCollisions: number
  notes: ImportNotes
}

export interface INPXFile {
  path: string
  name: string
}

export interface ImportPreview {
  libraryRoot: string
  zipCount: number
  inpxFiles: INPXFile[]
  inpxPath: string
  inpxFileName: string
  fileVersion: string
  catalogVersion: string
  sameVersion: boolean
  hasCatalog: boolean
}

export interface CloseRequested {
  committed: boolean
}

export function emptyNotes(): ImportNotes {
  return {
    missingArchives: [],
    missingArchivesTotal: 0,
    unnamedGenres: [],
    unnamedGenresTotal: 0,
    skippedMalformed: 0,
    skippedNoLibid: 0,
    encodings: { utf8: 0, cp1251: 0 },
    genreNamesMapped: 0,
  }
}

export function isImportReport(value: ImportReport | null | undefined): value is ImportReport {
  return !!value && typeof value.id === 'number' && value.id > 0 && !!value.status
}
