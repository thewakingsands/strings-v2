import { useEffect, useState } from 'react'
import {
  type LanguageHits,
  probeLanguages,
  probeTargets,
  rankProbes,
} from './languageProbe'

/** Asks the languages a search left out whether they would have found anything. */
export function useLanguageProbe({
  keyword,
  queryLanguages,
  enabled,
}: {
  keyword: string
  queryLanguages: string[]
  enabled: boolean
}): { loading: boolean; ranked: LanguageHits[] } {
  const [loading, setLoading] = useState(false)
  const [ranked, setRanked] = useState<LanguageHits[]>([])
  // A string, so a new array with the same languages does not probe again.
  const languagesKey = queryLanguages.join(',')

  useEffect(() => {
    setRanked([])
    const targets = probeTargets(languagesKey.split(','))
    if (!enabled || !keyword || targets.length === 0) {
      setLoading(false)
      return
    }

    const abort = new AbortController()
    setLoading(true)
    probeLanguages(keyword, targets, abort.signal)
      .then((totals) => {
        if (abort.signal.aborted) return
        setRanked(rankProbes(totals))
        setLoading(false)
      })
      .catch(() => {
        // Only an abort gets here, and the effect that caused it has taken over.
      })
    return () => abort.abort()
  }, [keyword, languagesKey, enabled])

  return { loading, ranked }
}
