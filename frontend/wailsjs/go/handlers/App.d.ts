export interface Capabilities {
  os: string
  backdropFilter: boolean
  framelessOk: boolean
  effectiveEffects: string
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

export function Bootstrap(): Promise<Bootstrap>
export function SetLocale(arg1: string): Promise<void>
export function SetTheme(arg1: string): Promise<void>
export function SetVisualEffects(arg1: string): Promise<void>
