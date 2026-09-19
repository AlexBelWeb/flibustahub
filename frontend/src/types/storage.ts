export interface DumpOffer {
  path: string
  name: string
  fileVersion?: string
  catalogVersion?: string
}

export interface StorageSnapshot {
  configured: boolean
  available: boolean
  unreachable: boolean
  libraryRoot?: string
  remapped?: boolean
  dumpOffer?: DumpOffer | null
}

export interface FileResult {
  path: string
  fileName: string
}

export function emptyStorage(): StorageSnapshot {
  return { configured: false, available: false, unreachable: false }
}
