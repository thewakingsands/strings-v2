import { describe, expect, it } from 'vitest'
import { probeTargets, rankProbes } from './languageProbe'

describe('probeTargets', () => {
  it('leaves out the languages already searched, in the fixed order', () => {
    expect(probeTargets(['ja', 'chs'])).toEqual(['tc', 'en', 'ko', 'de', 'fr'])
  })

  it('has nothing left when every language was searched', () => {
    expect(probeTargets(['chs', 'tc', 'en', 'ja', 'ko', 'de', 'fr'])).toEqual(
      [],
    )
  })
})

describe('rankProbes', () => {
  it('drops the languages without hits and puts the most hits first', () => {
    expect(
      rankProbes({ tc: 0, en: 4586, ja: 2, ko: 0, de: 2658, fr: 0 }),
    ).toEqual([
      { lang: 'en', total: 4586 },
      { lang: 'de', total: 2658 },
      { lang: 'ja', total: 2 },
    ])
  })

  it('breaks a tie by the fixed language order', () => {
    expect(rankProbes({ fr: 3, ja: 3, de: 3 })).toEqual([
      { lang: 'ja', total: 3 },
      { lang: 'de', total: 3 },
      { lang: 'fr', total: 3 },
    ])
  })

  it('is empty when nothing has hits', () => {
    expect(rankProbes({ en: 0, ja: 0 })).toEqual([])
  })
})
