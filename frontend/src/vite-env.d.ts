/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<object, object, unknown>
  export default component
}

interface Window {
  go: {
    handlers: {
      App: {
        Bootstrap: () => Promise<import('./types/bootstrap').Bootstrap>
        SetLocale: (code: string) => Promise<void>
        SetTheme: (theme: string) => Promise<void>
        SetVisualEffects: (mode: string) => Promise<void>
      }
    }
  }
}
