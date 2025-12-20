import { useEffect, useState } from 'react'
import type { StringItem } from './interface'
import { type IFileLineProps, linesByFile } from './linesByFile'
import { type IKeywordProps, linesByKeyword } from './linesByKeyword'

export interface ISearchQuery {
  keyword?: IKeywordProps
  file?: IFileLineProps
}

export function useSearch(initialQuery: ISearchQuery | null | undefined) {
  const [result, setResult] = useState<StringItem[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<Error | null>(null)
  const [query, setQuery] = useState<ISearchQuery | null | undefined>(
    initialQuery,
  )

  useEffect(() => {
    const abort = new AbortController()

    const fetchData = async () => {
      try {
        setError(null)
        setIsLoading(true)
        if (query) {
          if (query.keyword) {
            setResult(await linesByKeyword(query.keyword, abort.signal))
          } else if (query.file) {
            setResult(await linesByFile(query.file, abort.signal))
          } else {
            setResult([])
          }
        }
      } catch (e) {
        if (e && e instanceof Error && e.name !== 'AbortError') {
          setError(e)
        }
      } finally {
        if (!abort.signal.aborted) {
          setIsLoading(false)
        }
      }
    }

    fetchData()

    return () => abort.abort()
  }, [query])

  const setSearch = (query: ISearchQuery | null | undefined) => {
    setQuery(query)
  }

  const setPage = (page: number) => {
    if (query?.keyword) {
      setQuery({ keyword: { ...query.keyword, page: page } })
    }
  }

  return { result, query, isLoading, error, setSearch, setPage }
}
