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

export function formatQueryLanguagesLabel(languages: string[]): string {
  if (languages.length === 0) return '选择查询语言'
  if (languages.length === 1) {
    const lang = languages[0]
    return languageMap[lang as keyof typeof languageMap] ?? lang
  }
  return `搜索语言 (${languages.length})`
}

/** Sorts to the fixed languageMap order, for display columns where order is not a preference. */
export function sortToDisplayOrder(languages: string[]): string[] {
  const order = languageOptions.map((option) => option.value)
  return order.filter((lang) => languages.includes(lang))
}
