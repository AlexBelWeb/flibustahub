/// <reference types="vite/client" />

import type { Bootstrap } from './types/bootstrap'
import type {
  Author,
  AuthorPage,
  Genre,
  ListPeopleQuery,
  ListWorksQuery,
  SearchQuery,
  SearchResult,
  Series,
  SeriesPage,
  Work,
  WorkPage,
} from './types/catalog'
import type { ImportPreview, ImportReport } from './types/import'

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
        Bootstrap: () => Promise<Bootstrap>
        RetryStartup: () => Promise<Bootstrap>
        OpenLogsDir: () => Promise<void>
        OpenDataDir: () => Promise<void>
        SetLocale: (code: string) => Promise<void>
        SetTheme: (theme: string) => Promise<void>
        SetVisualEffects: (mode: string) => Promise<void>
        SetSidebarCollapsed: (collapsed: boolean) => Promise<void>
        SetCatalogView: (view: string) => Promise<void>
        SelectLibraryRoot: (title: string) => Promise<string>
        SelectINPXFile: (title: string) => Promise<string>
        SetINPXPath: (path: string) => Promise<void>
        PreviewImport: () => Promise<ImportPreview>
        StartImport: () => Promise<ImportReport>
        CancelImport: () => Promise<void>
        LastImportReport: () => Promise<ImportReport>
        DismissWindowClose: () => Promise<void>
        ListWorks: (query: ListWorksQuery) => Promise<WorkPage>
        SearchCatalog: (query: SearchQuery) => Promise<SearchResult>
        ListAuthors: (query: ListPeopleQuery) => Promise<AuthorPage>
        ListSeries: (query: ListPeopleQuery) => Promise<SeriesPage>
        ListGenres: (query: string) => Promise<Genre[]>
        GetWork: (id: number) => Promise<Work>
        GetAuthor: (id: number) => Promise<Author>
        GetGenre: (id: number) => Promise<Genre>
        GetSeries: (id: number) => Promise<Series>
        RandomWork: () => Promise<Work>
        CatalogAlphabet: () => Promise<string[]>
        RecordSearch: (query: string) => Promise<void>
        SearchHistory: () => Promise<string[]>
        ClearSearchHistory: () => Promise<void>
      }
    }
  }
}
