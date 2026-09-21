export interface DiagnosticPaths {
  dataDir: string
  dbPath: string
  coversDir: string
  logsDir: string
  backupsDir: string
  downloadsDir: string
  configPath: string
}

export interface DiagnosticSnapshot {
  version: string
  commit: string
  buildDate: string
  os: string
  arch: string
  webView: string
  inpxVersion: string
  works: number
  authors: number
  importedAt: string
  unnamedGenres: number
  aiProvider: string
  paths: DiagnosticPaths
  recentLogLines: string[]
}

export interface ArchiveResult {
  path: string
}

export interface IssueReport {
  kind: string
  title: string
  body: string
  githubUrl: string
  urlFits: boolean
}

export type IssueKind = 'bug' | 'idea'
