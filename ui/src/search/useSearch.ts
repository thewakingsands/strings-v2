import { useEffect, useState } from 'react'
import { emptySearchResult, type SearchResult } from './interface'
import { type IFileLineProps, linesByFile } from './linesByFile'
import { type IKeywordProps, linesByKeyword } from './linesByKeyword'

export interface ISearchQuery {
  keyword?: IKeywordProps
  file?: IFileLineProps
}

export function useSearch(initialQuery: ISearchQuery | null | undefined) {
  // Kept with the query it answers: while the next answer loads, `query` is already the new one,
  // and a plain search's total must not be read as an advanced one's count.
  const [answer, setAnswer] = useState<{
    result: SearchResult
    query?: ISearchQuery | null
  }>({ result: emptySearchResult })
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<Error | null>(null)
  const [query, setQuery] = useState<ISearchQuery | null | undefined>(
    initialQuery,
  )

  useEffect(() => {
    const abort = new AbortController()
    const setResult = (result: SearchResult) => setAnswer({ result, query })

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
            setResult(emptySearchResult)
          }
        } else {
          // Going back to the landing page must not leave the last results on it.
          setResult(emptySearchResult)
        }
      } catch (e) {
        if (e && e instanceof Error && e.name !== 'AbortError') {
          // The rows before the error are not this query's, and the next load would show them.
          setResult(emptySearchResult)
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
    // In the same update as the query: otherwise one render pairs the new query with the
    // previous result, which is enough for the empty-result probes to fire and be cancelled.
    setIsLoading(true)
    setQuery(query)
  }

  const setPage = (page: number) => {
    if (query?.keyword) {
      setQuery({ keyword: { ...query.keyword, page: page } })
    }
  }

  return {
    result: answer.result,
    resultQuery: answer.query,
    query,
    isLoading,
    error,
    setSearch,
    setPage,
  }
}
