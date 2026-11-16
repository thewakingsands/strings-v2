export const languageMap = {
  chs: '简体中文',
  tc: '繁体中文',
  en: 'English',
  ja: '日本語',
  ko: '한국어',
  de: 'Deutsch',
  fr: 'Français',
}

export interface LanguageOption {
  value: string
  label: string
}

export const languageOptions: LanguageOption[] = Object.entries(
  languageMap,
).map(([value, label]) => ({
  value,
  label,
}))

export const defaultQueryLanguages = ['chs']
export const defaultDisplayLanguages = ['chs', 'tc', 'en', 'ja'] as const

/**
 * Both language lists put the picked languages first, in whatever
 * order that group carries — priority for the query languages, column order for the display ones —
 * and everything else after them in the fixed languageMap order.
 */
export function orderActiveFirst(active: string[], all: string[]): string[] {
  const known = active.filter((lang) => all.includes(lang))
  return [...known, ...all.filter((lang) => !known.includes(lang))]
}

// The page shares its origin with a third party analytics script that keeps its own
// keys here, so both of ours carry a prefix.
const queryLanguagesKey = 'xivstrings.queryLanguages'
const displayLanguagesKey = 'xivstrings.displayLanguages'

const knownLanguages = new Set(Object.keys(languageMap))

/** Keeps the codes the app still knows, in the order given, without duplicates. */
function sanitizeLanguages(value: unknown): string[] | null {
  if (!Array.isArray(value)) return null
  const languages: string[] = []
  for (const lang of value) {
    if (typeof lang !== 'string') continue
    if (!knownLanguages.has(lang)) continue
    if (languages.includes(lang)) continue
    languages.push(lang)
  }
  return languages
}

function readLanguages(key: string): string[] | null {
  try {
    const raw = localStorage.getItem(key)
    if (raw === null) return null
    return sanitizeLanguages(JSON.parse(raw))
  } catch {
    // Either the value does not parse or storage itself is unavailable (private mode,
    // storage turned off). Neither may stop the app from starting, so fall back.
    return null
  }
}

function writeLanguages(key: string, languages: string[]): void {
  try {
    localStorage.setItem(key, JSON.stringify(languages))
  } catch {
    // A full or unavailable store must not undo the selection the user just made.
  }
}

export function readStoredQueryLanguages(): string[] {
  const stored = readLanguages(queryLanguagesKey)
  // The picker refuses to remove the last language and the server answers 400 without
  // one, so an empty list here can only have come from a corrupted value.
  if (!stored || stored.length === 0) return [...defaultQueryLanguages]
  return stored
}

export function readStoredDisplayLanguages(): string[] {
  const stored = readLanguages(displayLanguagesKey)
  if (!stored) return [...defaultDisplayLanguages]
  // Returned in the order it was stored, never re-sorted: STR-23 makes the display
  // order the one the user dragged the result columns into, so normalising it here
  // would quietly undo that. Validation drops what is unusable, nothing more.
  //
  // Empty stays empty too: unlike the query picker this one lets every language go,
  // which makes it a choice to restore rather than a corrupted value to replace.
  return stored
}

export function writeStoredQueryLanguages(languages: string[]): void {
  writeLanguages(queryLanguagesKey, languages)
}

export function writeStoredDisplayLanguages(languages: string[]): void {
  writeLanguages(displayLanguagesKey, languages)
}
