import { defineStore } from 'pinia'
import { ref } from 'vue'
import { parseBackendError, type BackendError } from '@/lib/backend-error'
import {
  asSearchResult,
  asWorkPage,
  type Author,
  type AuthorPage,
  type Genre,
  type ListPeopleQuery,
  type ListWorksQuery,
  type SearchQuery,
  type SearchResult,
  type Series,
  type SeriesPage,
  type Work,
  type WorkPage,
} from '@/types/catalog'

export type ListStatus = 'idle' | 'loading' | 'empty' | 'error' | 'ready' | 'missing'

export const HOME_ARRIVALS_KEY = 'home:added'

export interface WorkListState {
  items: Work[]
  nextCursor: string
  offset: number
  totalN?: number
  capped?: boolean
  fallback: boolean
  exhausted: boolean
  loadingMore: boolean
  status: ListStatus
  error: BackendError | null
  scrollTop: number
}

function emptyWorks(): WorkListState {
  return {
    items: [],
    nextCursor: '',
    offset: 0,
    fallback: false,
    exhausted: false,
    loadingMore: false,
    status: 'idle',
    error: null,
    scrollTop: 0,
  }
}

export interface PeopleListState<T> {
  items: T[]
  nextCursor: string
  exhausted: boolean
  loadingMore: boolean
  status: ListStatus
  error: BackendError | null
  scrollTop: number
}

function emptyPeople<T>(): PeopleListState<T> {
  return {
    items: [],
    nextCursor: '',
    exhausted: false,
    loadingMore: false,
    status: 'idle',
    error: null,
    scrollTop: 0,
  }
}

export const useCatalogStore = defineStore('catalog', () => {
  const works = ref<Record<string, WorkListState>>({})
  const authors = ref<Record<string, PeopleListState<Author>>>({})
  const series = ref<Record<string, PeopleListState<Series>>>({})
  const genres = ref<Record<string, PeopleListState<Genre>>>({})
  const alphabet = ref<string[]>([])
  const tokens: Record<string, number> = {}

  function nextToken(key: string): number {
    tokens[key] = (tokens[key] ?? 0) + 1
    return tokens[key]
  }

  function workState(key: string): WorkListState {
    if (!works.value[key]) {
      works.value[key] = emptyWorks()
    }
    return works.value[key]
  }

  function authorState(key: string): PeopleListState<Author> {
    if (!authors.value[key]) {
      authors.value[key] = emptyPeople()
    }
    return authors.value[key]
  }

  function seriesState(key: string): PeopleListState<Series> {
    if (!series.value[key]) {
      series.value[key] = emptyPeople()
    }
    return series.value[key]
  }

  function genreState(key: string): PeopleListState<Genre> {
    if (!genres.value[key]) {
      genres.value[key] = emptyPeople()
    }
    return genres.value[key]
  }

  function rememberScroll(key: string, top: number) {
    workState(key).scrollTop = top
  }

  function rememberPeopleScroll(kind: 'authors' | 'series' | 'genres', key: string, top: number) {
    if (kind === 'authors') {
      authorState(key).scrollTop = top
    } else if (kind === 'series') {
      seriesState(key).scrollTop = top
    } else {
      genreState(key).scrollTop = top
    }
  }

  function reset() {
    works.value = {}
    authors.value = {}
    series.value = {}
    genres.value = {}
  }

  async function loadAlphabet() {
    if (alphabet.value.length > 0) {
      return
    }
    alphabet.value = await window.go.handlers.App.CatalogAlphabet()
  }

  async function loadWorks(key: string, query: ListWorksQuery, resetPage: boolean) {
    const state = workState(key)
    if (resetPage) {
      state.items = []
      state.nextCursor = ''
      state.offset = 0
      state.exhausted = false
      state.loadingMore = false
      state.status = 'loading'
      state.error = null
    } else if (state.loadingMore || state.status === 'loading' || state.exhausted) {
      return
    } else {
      state.loadingMore = true
    }
    const token = nextToken(key)
    try {
      const page: WorkPage = asWorkPage(
        await window.go.handlers.App.ListWorks({
          ...query,
          cursor: resetPage ? '' : state.nextCursor,
        }),
      )
      if (tokens[key] !== token) {
        return
      }
      state.items = resetPage ? page.items : state.items.concat(page.items)
      state.nextCursor = page.nextCursor ?? ''
      state.exhausted = !state.nextCursor
      state.totalN = page.total?.n
      state.capped = Boolean(page.total?.capped)
      state.status = state.items.length === 0 ? 'empty' : 'ready'
    } catch (err) {
      if (tokens[key] !== token) {
        return
      }
      state.error = parseBackendError(err)
      state.status = state.items.length === 0 ? 'error' : 'ready'
    } finally {
      if (tokens[key] === token) {
        state.loadingMore = false
      }
    }
  }

  async function loadSearch(
    key: string,
    query: SearchQuery,
    resetPage: boolean,
  ): Promise<SearchResult> {
    const state = workState(key)
    if (resetPage) {
      state.items = []
      state.offset = 0
      state.exhausted = false
      state.loadingMore = false
      state.status = 'loading'
      state.error = null
      state.fallback = false
    } else if (state.loadingMore || state.status === 'loading' || state.exhausted) {
      return { works: { items: state.items }, fallback: state.fallback }
    } else {
      state.loadingMore = true
    }
    const token = nextToken(key)
    try {
      const result = asSearchResult(
        await window.go.handlers.App.SearchCatalog({
          ...query,
          offset: resetPage ? 0 : state.offset,
        }),
      )
      if (tokens[key] !== token) {
        return result
      }
      const page = result.works
      state.items = resetPage ? page.items : state.items.concat(page.items)
      state.offset = state.items.length
      state.exhausted = page.items.length === 0 || state.items.length >= 500
      state.totalN = page.total?.n
      state.capped = Boolean(page.total?.capped) || state.items.length >= 500
      state.fallback = Boolean(result.fallback)
      state.status = state.items.length === 0 ? 'empty' : 'ready'
      return result
    } catch (err) {
      if (tokens[key] !== token) {
        return { works: { items: state.items }, fallback: state.fallback }
      }
      state.error = parseBackendError(err)
      state.status = state.items.length === 0 ? 'error' : 'ready'
      return { works: { items: state.items }, fallback: state.fallback }
    } finally {
      if (tokens[key] === token) {
        state.loadingMore = false
      }
    }
  }

  async function loadAuthors(key: string, query: ListPeopleQuery, resetPage: boolean) {
    const state = authorState(key)
    if (resetPage) {
      state.items = []
      state.nextCursor = ''
      state.exhausted = false
      state.status = 'loading'
      state.error = null
    } else if (state.loadingMore || state.status === 'loading' || state.exhausted) {
      return
    } else {
      state.loadingMore = true
    }
    const token = nextToken(key)
    try {
      const page: AuthorPage = await window.go.handlers.App.ListAuthors({
        ...query,
        cursor: resetPage ? '' : state.nextCursor,
      })
      if (tokens[key] !== token) {
        return
      }
      const items = page.items ?? []
      state.items = resetPage ? items : state.items.concat(items)
      state.nextCursor = page.nextCursor ?? ''
      state.exhausted = !state.nextCursor
      state.status = state.items.length === 0 ? 'empty' : 'ready'
    } catch (err) {
      if (tokens[key] !== token) {
        return
      }
      state.error = parseBackendError(err)
      state.status = state.items.length === 0 ? 'error' : 'ready'
    } finally {
      if (tokens[key] === token) {
        state.loadingMore = false
      }
    }
  }

  async function loadSeriesList(key: string, query: ListPeopleQuery, resetPage: boolean) {
    const state = seriesState(key)
    if (resetPage) {
      state.items = []
      state.nextCursor = ''
      state.exhausted = false
      state.status = 'loading'
      state.error = null
    } else if (state.loadingMore || state.status === 'loading' || state.exhausted) {
      return
    } else {
      state.loadingMore = true
    }
    const token = nextToken(key)
    try {
      const page: SeriesPage = await window.go.handlers.App.ListSeries({
        ...query,
        cursor: resetPage ? '' : state.nextCursor,
      })
      if (tokens[key] !== token) {
        return
      }
      const items = page.items ?? []
      state.items = resetPage ? items : state.items.concat(items)
      state.nextCursor = page.nextCursor ?? ''
      state.exhausted = !state.nextCursor
      state.status = state.items.length === 0 ? 'empty' : 'ready'
    } catch (err) {
      if (tokens[key] !== token) {
        return
      }
      state.error = parseBackendError(err)
      state.status = state.items.length === 0 ? 'error' : 'ready'
    } finally {
      if (tokens[key] === token) {
        state.loadingMore = false
      }
    }
  }

  async function loadGenres(key: string, query: string) {
    const state = genreState(key)
    state.status = 'loading'
    state.error = null
    const token = nextToken(key)
    try {
      const items = (await window.go.handlers.App.ListGenres(query)) ?? []
      if (tokens[key] !== token) {
        return
      }
      state.items = items
      state.exhausted = true
      state.status = items.length === 0 ? 'empty' : 'ready'
    } catch (err) {
      if (tokens[key] !== token) {
        return
      }
      state.error = parseBackendError(err)
      state.status = 'error'
    }
  }

  async function getWork(id: number): Promise<Work> {
    return window.go.handlers.App.GetWork(id)
  }

  async function getAuthor(id: number): Promise<Author> {
    return window.go.handlers.App.GetAuthor(id)
  }

  async function getGenre(id: number) {
    return window.go.handlers.App.GetGenre(id)
  }

  async function getSeries(id: number): Promise<Series> {
    return window.go.handlers.App.GetSeries(id)
  }

  async function randomWork(): Promise<Work> {
    return window.go.handlers.App.RandomWork()
  }

  return {
    works,
    authors,
    series,
    genres,
    alphabet,
    workState,
    authorState,
    seriesState,
    genreState,
    rememberScroll,
    rememberPeopleScroll,
    reset,
    loadAlphabet,
    loadWorks,
    loadSearch,
    loadAuthors,
    loadSeriesList,
    loadGenres,
    getWork,
    getAuthor,
    getGenre,
    getSeries,
    randomWork,
  }
})
