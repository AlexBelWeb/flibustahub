/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<object, object, unknown>
  export default component
}

interface Window {
  runtime?: {
    EventsOn: (eventName: string, callback: (...data: unknown[]) => void) => () => void
    EventsOff: (eventName: string) => void
    Quit: () => void
  }
  go: {
    handlers: {
      App: {
        Bootstrap: () => Promise<import('./types/bootstrap').Bootstrap>
        RetryStartup: () => Promise<import('./types/bootstrap').Bootstrap>
        OpenLogsDir: () => Promise<void>
        OpenDataDir: () => Promise<void>
        SetLocale: (code: string) => Promise<void>
        SetTheme: (theme: string) => Promise<void>
        SetVisualEffects: (mode: string) => Promise<void>
        SelectLibraryRoot: (title: string) => Promise<string>
        PreviewImport: () => Promise<import('./types/import').ImportPreview>
        StartImport: () => Promise<import('./types/import').ImportReport>
        CancelImport: () => Promise<void>
        LastImportReport: () => Promise<import('./types/import').ImportReport>
        DismissWindowClose: () => Promise<void>
      }
    }
  }
}
