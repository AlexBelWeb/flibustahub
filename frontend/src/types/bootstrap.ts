export interface Capabilities {
  os: string
  backdropFilter: boolean
  framelessOk: boolean
  effectiveEffects: 'full' | 'reduced' | string
}

export interface Paths {
  dataDir: string
  dbPath: string
  coversDir: string
  logsDir: string
  backupsDir: string
  downloadsDir: string
  configPath: string
}

export interface Bootstrap {
  version: string
  commit: string
  buildDate: string
  locale: string
  theme: string
  visualEffectsPref: string
  capabilities: Capabilities
  libraryRoot: string
  paths: Paths
}
