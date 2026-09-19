import type { LocationQuery } from 'vue-router'
import { queryId } from '@/lib/route-query'

export function withWorkQuery(query: LocationQuery, workId: number): LocationQuery {
  return { ...query, work: String(workId) }
}

export function withoutWorkQuery(query: LocationQuery): LocationQuery {
  const next = { ...query }
  delete next.work
  return next
}

export function workIdFromQuery(query: LocationQuery): number {
  return queryId(query.work)
}
