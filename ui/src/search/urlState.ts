import { sanitizeLanguages } from '../utils/language'
import type { LanguageSelection } from '../utils/useLanguageSelection'
import type { IKeywordProps } from './linesByKeyword'
import type { ISearchQuery } from './useSearch'

/** Rows shown on each side of the one a context view is opened at. */
export const CONTEXT_RADIUS = 20

/** What the address bar says. A language group is absent when the URL does not carry it. */
export type UrlView =
  | { kind: 'home' }
  | {
      kind: 'search'
      keyword: string
      advanced: boolean
      page: number
      queryLanguages?: string[]
      displayLanguages?: string[]
    }
  | {
      kind: 'context'
      sheet: string
      index: number
      displayLanguages?: string[]
    }

/** A view with every language group filled in, which is what a written URL must carry. */
export type ResolvedUrlView =
  | { kind: 'home' }
  | {
      kind: 'search'
      keyword: string
      advanced: boolean
      page: number
      queryLanguages: string[]
      displayLanguages: string[]
    }
  | {
      kind: 'context'
      sheet: string
      index: number
      displayLanguages: string[]
    }

function parseLanguages(value: string | null): string[] | undefined {
  if (value === null) return undefined
  const languages = sanitizeLanguages(value.split(','))
  // An empty list is the user's own choice only when it comes from their storage; in a link it
  // can only be a broken one, and an empty lang would be answered with a 400.
  return languages?.length ? languages : undefined
}

function parseNonNegative(value: string | null): number | null {
  if (value === null || !/^\d+$/.test(value)) return null
  const n = Number(value)
  return Number.isSafeInteger(n) ? n : null
}

export function parseUrlView(search: string): UrlView {
  const params = new URLSearchParams(search)
  const displayLanguages = parseLanguages(params.get('show'))

  const sheet = params.get('sheet')
  const at = parseNonNegative(params.get('at'))
  if (sheet && at !== null) {
    return { kind: 'context', sheet, index: at, displayLanguages }
  }

  const keyword = params.get('q') ?? ''
  if (!keyword.trim()) return { kind: 'home' }

  const page = parseNonNegative(params.get('page'))
  return {
    kind: 'search',
    keyword,
    advanced: params.get('mode') === 'advanced',
    page: page && page > 0 ? page : 1,
    queryLanguages: parseLanguages(params.get('lang')),
    displayLanguages,
  }
}

export function resolveUrlView(
  view: UrlView,
  fallback: LanguageSelection,
): ResolvedUrlView {
  switch (view.kind) {
    case 'home':
      return view
    case 'search':
      return {
        ...view,
        queryLanguages: view.queryLanguages ?? fallback.queryLanguages,
        displayLanguages: view.displayLanguages ?? fallback.displayLanguages,
      }
    case 'context':
      return {
        ...view,
        displayLanguages: view.displayLanguages ?? fallback.displayLanguages,
      }
  }
}

/** The keys go out in a fixed order, so one search always gives one link. */
export function formatUrlView(view: ResolvedUrlView): string {
  const params = new URLSearchParams()
  switch (view.kind) {
    case 'home':
      return ''
    case 'search':
      params.set('q', view.keyword)
      params.set('lang', view.queryLanguages.join(','))
      params.set('show', view.displayLanguages.join(','))
      if (view.advanced) params.set('mode', 'advanced')
      if (view.page > 1) params.set('page', String(view.page))
      break
    case 'context':
      params.set('sheet', view.sheet)
      params.set('at', String(view.index))
      params.set('show', view.displayLanguages.join(','))
      break
  }
  // A comma means nothing special in a query string, so the escape only makes a shared link
  // harder to read.
  return `?${params.toString().replaceAll('%2C', ',')}`
}

export function searchView(
  keyword: IKeywordProps,
  displayLanguages: string[],
): ResolvedUrlView {
  return {
    kind: 'search',
    keyword: keyword.keyword,
    advanced: keyword.advanced ?? false,
    page: keyword.page,
    queryLanguages: keyword.languages,
    displayLanguages,
  }
}

export function viewToQuery(
  view: ResolvedUrlView,
  pageSize: number,
): ISearchQuery | undefined {
  switch (view.kind) {
    case 'home':
      return undefined
    case 'search':
      return {
        keyword: {
          keyword: view.keyword,
          page: view.page,
          pageSize,
          languages: view.queryLanguages,
          displayLanguages: view.displayLanguages,
          advanced: view.advanced,
        },
      }
    case 'context':
      return {
        file: {
          sheet: view.sheet,
          indexLower: Math.max(0, view.index - CONTEXT_RADIUS),
          indexHigher: view.index + CONTEXT_RADIUS,
          displayLanguages: view.displayLanguages,
        },
      }
  }
}
