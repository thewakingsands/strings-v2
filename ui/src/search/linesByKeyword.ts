import { emptySearchResult, type SearchResult } from './interface'
import { searchApi } from './query'

export interface IKeywordProps {
  keyword: string
  pageSize: number
  page: number
  /** Ordered by decreasing priority: the server boosts earlier languages higher. */
  languages: string[]
  displayLanguages?: string[]
  /** Reads the keyword as a bleve query string instead of plain words. */
  advanced?: boolean
}

export async function linesByKeyword(
  {
    keyword,
    pageSize,
    page,
    languages,
    displayLanguages,
    advanced,
  }: IKeywordProps,
  signal?: AbortSignal,
): Promise<SearchResult> {
  if (!keyword || languages.length === 0) {
    return emptySearchResult
  }

  const response = await searchApi(
    {
      langs: languages,
      q: keyword,
      advanced,
      offset: (page - 1) * pageSize,
      limit: pageSize,
      fields: displayLanguages,
    },
    signal,
  )

  return {
    items: response.data,
    total: response.meta.total,
  }
}
