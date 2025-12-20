import type { SearchResult } from './interface'
import { itemsApi } from './query'

export interface IFileLineProps {
  sheet: string
  indexLower: number
  indexHigher: number
}

export async function linesByFile(
  props: IFileLineProps,
  signal?: AbortSignal,
): Promise<SearchResult> {
  const sheet = props.sheet.replace(/\.json$/, '')
  const limit = props.indexHigher - props.indexLower + 1
  const offset = props.indexLower

  const response = await itemsApi(
    {
      sheet,
      offset,
      limit,
    },
    signal,
  )

  return {
    items: response.data,
    total: response.meta.total,
  }
}
