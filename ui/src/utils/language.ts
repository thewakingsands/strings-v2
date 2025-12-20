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

export const defaultLanguage = 'chs'
export const defaultDisplayLanguages = ['chs', 'tc', 'en', 'ja'] as const
