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
  // A string, so a new array with the same languages does not probe again.
  const languagesKey = queryLanguages.join(',')
  const probeKey = `${keyword}\n${languagesKey}`
  const [settled, setSettled] = useState<{
    key: string
    ranked: LanguageHits[]
  } | null>(null)
  const willProbe =
    enabled && !!keyword && probeTargets(queryLanguages).length > 0

  useEffect(() => {
    const targets = probeTargets(languagesKey.split(','))
    if (!enabled || !keyword || targets.length === 0) return

    const abort = new AbortController()
    probeLanguages(keyword, targets, abort.signal)
      .then((totals) => {
        if (abort.signal.aborted) return
        setSettled({
          key: `${keyword}\n${languagesKey}`,
          ranked: rankProbes(totals),
        })
      })
      .catch(() => {
        // Only an abort gets here, and the effect that caused it has taken over.
      })
    return () => abort.abort()
  }, [keyword, languagesKey, enabled])

  // Derived rather than set from the effect: the render that first enables the probe comes
  // before the effect, and a loading flag still false there would read as "nothing anywhere".
  const current = settled?.key === probeKey ? settled.ranked : null
  return {
    loading: willProbe && current === null,
    ranked: willProbe ? (current ?? []) : [],
  }
}
