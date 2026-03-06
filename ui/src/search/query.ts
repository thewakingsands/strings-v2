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

const sheetPrefix = 'filename:'

function processQuery(searchParams: URLSearchParams, q: string) {
  const parts = q.split(' ')

  // Extract sheet from query
  const sheetPart = parts.findIndex((part) => part.startsWith(sheetPrefix))
  if (sheetPart !== -1) {
    searchParams.set('sheet', parts[sheetPart].slice(sheetPrefix.length))
    parts.splice(sheetPart, 1)
  }

  searchParams.set('q', parts.filter((part) => !!part).join(' '))
}

export async function searchApi(
  params: {
    lang: string
    q: string
    offset?: number
    limit?: number
    fields?: string[]
  },
  signal?: AbortSignal,
): Promise<ApiResponse<StringItem[]>> {
  const searchParams = new URLSearchParams()
  searchParams.set('lang', params.lang)
  processQuery(searchParams, params.q)

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
    throw new Error(error.error || `HTTP ${resp.status}`)
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
    throw new Error(error.error || `HTTP ${resp.status}`)
  }

  return resp.json()
}
