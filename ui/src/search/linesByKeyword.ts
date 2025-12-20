import type { StringItem } from './interface'
import { searchApi } from './query'

export interface IKeywordProps {
  keyword: string
  pageSize: number
  page: number
  language: string
}

export async function linesByKeyword(
  { keyword, pageSize, page, language }: IKeywordProps,
  signal?: AbortSignal,
): Promise<StringItem[]> {
  if (!keyword) {
    return []
  }

  // Search in the selected language only
  const response = await searchApi(
    {
      lang: language,
      q: keyword,
      offset: (page - 1) * pageSize,
      limit: pageSize,
    },
    signal,
  )

  return response.data
}
