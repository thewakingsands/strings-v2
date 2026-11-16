import type { StringItem } from './interface'

export interface ApiResponse<T> {
  data: T
  meta: {
    elapsed: string
    total: number
  }
}

export interface ApiError {
  error: string
}

/** Keeps the status so the error state can tell a rejected query from a failing server. */
export class HttpError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

const sheetPrefix = 'filename:'
const excludedSheetPrefix = '-filename:'

/**
 * Takes every `filename:` and `-filename:` token out of q and turns them into the
 * sheet parameter: `覚えていろ filename:quest/* -filename:Item,Action` becomes
 * q `覚えていろ` and sheet `quest/*,-Item,-Action`. An empty sheet means no filter.
 */
export function splitSheetTokens(q: string): { q: string; sheet: string } {
  const words: string[] = []
  const entries: string[] = []

  for (const token of q.split(/\s+/)) {
    if (token.startsWith(sheetPrefix)) {
      entries.push(...sheetParts(token.slice(sheetPrefix.length)))
    } else if (token.startsWith(excludedSheetPrefix)) {
      entries.push(
        ...sheetParts(token.slice(excludedSheetPrefix.length)).map(
          (part) => `-${part}`,
        ),
      )
    } else if (token) {
      words.push(token)
    }
  }

  return { q: words.join(' '), sheet: entries.join(',') }
}

function sheetParts(value: string): string[] {
  return value.split(',').filter((part) => !!part)
}

export async function searchApi(
  params: {
    /** Ordered by decreasing priority; sent as a comma separated lang parameter. */
    langs: string[]
    q: string
    /** Reads q as a bleve query string instead of plain words. */
    advanced?: boolean
    offset?: number
    limit?: number
    fields?: string[]
  },
  signal?: AbortSignal,
): Promise<ApiResponse<StringItem[]>> {
  const searchParams = new URLSearchParams()
  searchParams.set('lang', params.langs.join(','))
  const { q, sheet } = splitSheetTokens(params.q)
  searchParams.set('q', q)
  if (sheet) {
    searchParams.set('sheet', sheet)
  }
  if (params.advanced) {
    searchParams.set('mode', 'advanced')
  }

  if (params.offset !== undefined) {
    searchParams.set('offset', params.offset.toString())
  }
  if (params.limit !== undefined) {
    searchParams.set('limit', params.limit.toString())
  }
  if (params.fields && params.fields.length > 0) {
    searchParams.append('fields', params.fields.join(','))
  }

  const resp = await fetch(`/api/search?${searchParams.toString()}`, {
    method: 'GET',
    signal,
  })

  if (!resp.ok) {
    const error: ApiError = await resp.json()
    throw new HttpError(resp.status, error.error || `HTTP ${resp.status}`)
  }

  return resp.json()
}

export async function itemsApi(
  params: {
    sheet: string
    offset?: number
    limit?: number
    fields?: string[]
  },
  signal?: AbortSignal,
): Promise<ApiResponse<StringItem[]>> {
  const searchParams = new URLSearchParams()
  searchParams.set('sheet', params.sheet)
  if (params.offset !== undefined) {
    searchParams.set('offset', params.offset.toString())
  }
  if (params.limit !== undefined) {
    searchParams.set('limit', params.limit.toString())
  }
  if (params.fields && params.fields.length > 0) {
    searchParams.append('fields', params.fields.join(','))
  }

  const resp = await fetch(`/api/items?${searchParams.toString()}`, {
    method: 'GET',
    signal,
  })

  if (!resp.ok) {
    const error: ApiError = await resp.json()
    throw new HttpError(resp.status, error.error || `HTTP ${resp.status}`)
  }

  return resp.json()
}
