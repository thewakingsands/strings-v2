import { languageOptions } from '@/utils/language'
import { searchApi } from './query'

const order = languageOptions.map((option) => option.value)

/** The languages a zero-result search did not look in, in the fixed languageMap order. */
export function probeTargets(queryLanguages: string[]): string[] {
  return order.filter((lang) => !queryLanguages.includes(lang))
}

export interface LanguageHits {
  lang: string
  total: number
}

/** Only the languages that would find something, most hits first. */
export function rankProbes(totals: Record<string, number>): LanguageHits[] {
  return Object.entries(totals)
    .filter(([, total]) => total > 0)
    .map(([lang, total]) => ({ lang, total }))
    .sort(
      (a, b) =>
        b.total - a.total || order.indexOf(a.lang) - order.indexOf(b.lang),
    )
}

/** Counts the hits of keyword in each of langs alone. One failed language counts as none. */
export async function probeLanguages(
  keyword: string,
  langs: string[],
  signal: AbortSignal,
): Promise<Record<string, number>> {
  const totals = await Promise.all(
    langs.map(async (lang) => {
      try {
        const response = await searchApi(
          { langs: [lang], q: keyword, limit: 1, fields: [lang] },
          signal,
        )
        return [lang, response.meta.total] as const
      } catch (e) {
        if (signal.aborted) throw e
        return [lang, 0] as const
      }
    }),
  )
  return Object.fromEntries(totals)
}
