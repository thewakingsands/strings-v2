import { describe, expect, it } from 'vitest'
import {
  hideDisplay,
  type LanguageChangeReason,
  reasonNeedsRefetch,
  reorder,
  replaceQuery,
  showDisplay,
  toggleQuery,
} from './useLanguageSelection'

describe('query languages', () => {
  it('appends a new one, which puts it last in priority', () => {
    expect(toggleQuery(['chs'], 'ja')).toEqual(['chs', 'ja'])
  })

  it('removes one that is already picked', () => {
    expect(toggleQuery(['chs', 'ja'], 'chs')).toEqual(['ja'])
  })

  it('refuses to remove the last one, which the server answers 400 for', () => {
    expect(toggleQuery(['chs'], 'chs')).toBeNull()
  })
})

describe('replacing the query languages', () => {
  it('replaces the whole group, as switching the search to one language does', () => {
    expect(replaceQuery(['chs', 'ja'], ['en'])).toEqual(['en'])
  })

  it('refuses an empty group, which the server answers 400 for', () => {
    expect(replaceQuery(['chs'], [])).toBeNull()
  })

  it('refuses a group identical to the current one', () => {
    expect(replaceQuery(['chs', 'ja'], ['chs', 'ja'])).toBeNull()
  })
})

describe('display languages', () => {
  it('appends a new column at the end rather than sorting it in', () => {
    // The order is the one the user dragged the columns into, so it must survive an addition.
    expect(showDisplay(['ja', 'chs'], 'en')).toEqual(['ja', 'chs', 'en'])
  })

  it('ignores a language that already has a column', () => {
    expect(showDisplay(['chs', 'en'], 'en')).toBeNull()
  })

  it('hides a column', () => {
    expect(hideDisplay(['chs', 'en'], 'chs')).toEqual(['en'])
  })

  it('refuses to hide the last column', () => {
    expect(hideDisplay(['chs'], 'chs')).toBeNull()
  })

  it('ignores a language that has no column to hide', () => {
    expect(hideDisplay(['chs', 'en'], 'ja')).toBeNull()
  })
})

describe('the two groups do not touch each other', () => {
  it('adding a search language leaves the columns alone', () => {
    // STR-23 dropped the top-up that used to give every searched language a column.
    const display = ['chs', 'en']
    expect(toggleQuery(['chs'], 'ja')).toEqual(['chs', 'ja'])
    expect(display).toEqual(['chs', 'en'])
  })

  it('hiding the column of a language being searched is allowed', () => {
    expect(hideDisplay(['chs', 'ja'], 'ja')).toEqual(['chs'])
  })
})

describe('reorder', () => {
  it('moves an entry without changing the set', () => {
    expect(reorder(['chs', 'tc', 'en'], 2, 0)).toEqual(['en', 'chs', 'tc'])
  })

  it('answers null when nothing would move', () => {
    expect(reorder(['chs', 'tc'], 1, 1)).toBeNull()
  })

  it('answers null for an index outside the list', () => {
    expect(reorder(['chs', 'tc'], 0, 5)).toBeNull()
    expect(reorder(['chs', 'tc'], -1, 0)).toBeNull()
  })
})

describe('which changes need a new request', () => {
  const cases: Array<[LanguageChangeReason, boolean]> = [
    ['query', true],
    // The lang parameter's order is the priority the server boosts by, so this changes results.
    ['query-order', true],
    ['display', true],
    ['column-order', false],
    ['restore', false],
  ]

  for (const [reason, needed] of cases) {
    it(`${reason} → ${needed ? 'refetch' : 'no refetch'}`, () => {
      expect(reasonNeedsRefetch(reason)).toBe(needed)
    })
  }
})
