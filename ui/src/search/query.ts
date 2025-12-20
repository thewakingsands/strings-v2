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

export async function searchApi(
  params: {
    lang: string
    q: string
    sheet?: string
    offset?: number
    limit?: number
    fields?: string[]
  },
  signal?: AbortSignal,
): Promise<ApiResponse<StringItem[]>> {
  const searchParams = new URLSearchParams()
  searchParams.set('lang', params.lang)
  searchParams.set('q', params.q)
  if (params.sheet) {
    searchParams.set('sheet', params.sheet)
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
