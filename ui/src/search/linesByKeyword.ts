import { emptySearchResult, type SearchResult } from './interface'
import { searchApi } from './query'

export interface IKeywordProps {
  keyword: string
  pageSize: number
  page: number
  language: string
  displayLanguages?: string[]
}

export async function linesByKeyword(
  { keyword, pageSize, page, language, displayLanguages }: IKeywordProps,
  signal?: AbortSignal,
): Promise<SearchResult> {
  if (!keyword) {
    return emptySearchResult
  }

  // Search in the selected language only
  const response = await searchApi(
    {
      lang: language,
      q: keyword,
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
