import type { StringItem } from './interface'
import { itemsApi } from './query'

export interface IFileLineProps {
  sheet: string
  indexLower: number
  indexHigher: number
}

export async function linesByFile(
  props: IFileLineProps,
  signal?: AbortSignal,
): Promise<StringItem[]> {
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

  return response.data
}
