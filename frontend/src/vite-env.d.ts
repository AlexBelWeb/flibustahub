/// <reference types="vite/client" />

import type { Bootstrap } from './types/bootstrap'
import type {
  Author,
  AuthorPage,
  Genre,
  HomeDashboard,
  ListPeopleQuery,
  ListWorksQuery,
  SearchQuery,
  SearchResult,
  Series,
  SeriesPage,
  Work,
  WorkDetails,
  WorkPage,
  Annotation,
  CoverProgress,
} from './types/catalog'
import type { ImportPreview, ImportReport } from './types/import'
import type { PersonalExportResult, PersonalImportReport, PersonalSnapshot } from './types/personal'
import type { FileResult, StorageSnapshot } from './types/storage'

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
    BrowserOpenURL: (url: string) => void
  }
  go: {
    handlers: {
      App: {
        Bootstrap: () => Promise<Bootstrap>
        RetryStartup: () => Promise<Bootstrap>
        OpenLogsDir: () => Promise<void>
        OpenDataDir: () => Promise<void>
        OpenDownloadsDir: () => Promise<void>
        SetLocale: (code: string) => Promise<void>
        SetTheme: (theme: string) => Promise<void>
        SetVisualEffects: (mode: string) => Promise<void>
        SetSidebarCollapsed: (collapsed: boolean) => Promise<void>
        SetCatalogView: (view: string) => Promise<void>
        CheckStorage: (force: boolean) => Promise<StorageSnapshot>
        DismissDumpOffer: () => Promise<void>
        DownloadEdition: (id: number) => Promise<FileResult>
        ReadEdition: (id: number) => Promise<FileResult>
        CancelFileOp: (id: number, kind: string) => Promise<void>
        ShowInFolder: (path: string) => Promise<void>
        SelectDownloadsDir: (title: string) => Promise<string>
        SelectReaderPath: (title: string) => Promise<string>
        ClearReaderPath: () => Promise<void>
        ReaderPath: () => Promise<string>
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
        GetWorkDetails: (id: number) => Promise<WorkDetails>
        RecordViewed: (id: number) => Promise<void>
        GetAnnotation: (id: number) => Promise<Annotation>
        ClearCoverCache: () => Promise<void>
        CoverWarmupPreview: () => Promise<number>
        StartCoverWarmup: () => Promise<void>
        StopCoverWarmup: () => void
        CoverWarmupProgress: () => Promise<CoverProgress>
        GetAuthor: (id: number) => Promise<Author>
        GetGenre: (id: number) => Promise<Genre>
        GetSeries: (id: number) => Promise<Series>
        RandomWork: () => Promise<Work>
        CatalogAlphabet: () => Promise<string[]>
        RecordSearch: (query: string) => Promise<void>
        SearchHistory: () => Promise<string[]>
        ClearSearchHistory: () => Promise<void>
        GetHome: () => Promise<HomeDashboard>
        PersonalSnapshot: () => Promise<PersonalSnapshot>
        SetWorkRating: (id: number, rating: number) => Promise<void>
        SetWorkComment: (id: number, comment: string) => Promise<void>
        SetWorkWantToRead: (id: number, want: boolean) => Promise<void>
        ExportPersonal: (path: string) => Promise<PersonalExportResult>
        PreviewPersonalImport: (path: string) => Promise<PersonalImportReport>
        ImportPersonal: (path: string) => Promise<PersonalImportReport>
        SelectPersonalExportPath: (title: string) => Promise<string>
        SelectPersonalImportPath: (title: string) => Promise<string>
      }
    }
  }
}
