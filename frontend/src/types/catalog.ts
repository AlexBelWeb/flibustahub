export interface CatalogTotal {
  n: number
  capped?: boolean
}

export interface Work {
  id: number
  workKey: string
  title: string
  sortTitle: string
  authorsText: string
  lang?: string
  rating?: number
  wantToRead?: boolean
  addedDate?: string
  series?: string
  seriesNo?: string
  editionCount: number
  hasFile: boolean
  size?: number
  librate?: number
}

export interface Author {
  id: number
  displayName: string
  sortName: string
  workCount: number
}

export interface Series {
  id: number
  name: string
  sortName: string
  workCount: number
}

export interface Genre {
  id: number
  code: string
  nameRu: string
  workCount: number
}

export interface WorkDetails extends Work {
  comment?: string
  authors?: Author[]
  genres?: Genre[]
  seriesId?: number
  prevWorkId?: number
  nextWorkId?: number
  annotation?: string
  annotationChecked?: boolean
  fileExt?: string
  preferredEditionId?: number
  editions?: WorkEdition[]
}

export interface WorkEdition {
  id: number
  archiveName: string
  fileName?: string
  fileExt?: string
  size?: number
  addedDate?: string
  preferred?: boolean
}

export interface Annotation {
  text?: string
  checked: boolean
}

export interface CoverProgress {
  total: number
  done: number
  running: boolean
}

export interface WorkPage {
  items: Work[]
  nextCursor?: string
  total?: CatalogTotal | null
}

export interface AuthorPage {
  items: Author[]
  nextCursor?: string
  total?: CatalogTotal | null
}

export interface SeriesPage {
  items: Series[]
  nextCursor?: string
  total?: CatalogTotal | null
}

export interface ListWorksQuery {
  sort?: string
  cursor?: string
  lang?: string
  genreId?: number
  authorId?: number
  seriesId?: number
  rated?: boolean
  want?: boolean
  limit?: number
}

export interface ListPeopleQuery {
  letter?: string
  query?: string
  cursor?: string
  limit?: number
}

export interface SearchQuery {
  q: string
  lang?: string
  genreId?: number
  authorId?: number
  seriesId?: number
  rated?: boolean
  want?: boolean
  offset?: number
  limit?: number
}

export interface HomeDashboard {
  worksListable: number
  authorsTotal: number
  seriesTotal: number
  inpxVersion?: string
  importedAt?: string
  hero?: Work
  heroSource?: string
  arrivals?: Work[]
  rated?: Work[]
  wantToReadCount: number
  popularGenres?: Genre[]
  popularSeries?: Series[]
}

export interface SearchResult {
  authors?: Author[]
  series?: Series[]
  authorsTotal?: CatalogTotal
  seriesTotal?: CatalogTotal
  works: WorkPage
  fallback?: boolean
}

export type CatalogView = 'tile' | 'table'

export function isWork(value: unknown): value is Work {
  return !!value && typeof value === 'object' && typeof (value as Work).id === 'number'
}

export function asWorkPage(value: WorkPage | null | undefined): WorkPage {
  return {
    items: value?.items ?? [],
    nextCursor: value?.nextCursor,
    total: value?.total,
  }
}

export function asSearchResult(value: SearchResult | null | undefined): SearchResult {
  return {
    authors: value?.authors ?? [],
    series: value?.series ?? [],
    authorsTotal: value?.authorsTotal,
    seriesTotal: value?.seriesTotal,
    works: asWorkPage(value?.works),
    fallback: Boolean(value?.fallback),
  }
}

export function asHomeDashboard(value: HomeDashboard | null | undefined): HomeDashboard {
  return {
    worksListable: value?.worksListable ?? 0,
    authorsTotal: value?.authorsTotal ?? 0,
    seriesTotal: value?.seriesTotal ?? 0,
    inpxVersion: value?.inpxVersion,
    importedAt: value?.importedAt,
    hero: value?.hero,
    heroSource: value?.heroSource,
    arrivals: value?.arrivals ?? [],
    rated: value?.rated ?? [],
    wantToReadCount: value?.wantToReadCount ?? 0,
    popularGenres: value?.popularGenres ?? [],
    popularSeries: value?.popularSeries ?? [],
  }
}
