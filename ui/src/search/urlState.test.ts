import { describe, expect, it } from 'vitest'
import {
  formatUrlView,
  parseUrlView,
  resolveUrlView,
  searchView,
  viewToQuery,
} from './urlState'

const fallback = {
  queryLanguages: ['chs'],
  displayLanguages: ['chs', 'tc', 'en', 'ja'],
}

const roundTrip = (search: string) =>
  formatUrlView(resolveUrlView(parseUrlView(search), fallback))

describe('round trip', () => {
  it.each([
    '?q=%E8%A6%9A%E3%81%88%E3%81%A6%E3%81%84%E3%82%8D&lang=ja,chs&show=chs,tc,en,ja',
    '?q=a&lang=chs&show=ja,en&mode=advanced&page=2',
    '?sheet=quest%2F011%2FSubFst001_00012&at=123&show=chs,tc,en,ja',
    '',
  ])('leaves the canonical %s as it is', (search) => {
    expect(roundTrip(search)).toBe(search)
  })

  it.each([
    '覚えていろ',
    '覚えていろ filename:quest/*',
    'a b -filename:Item,Action',
    '50% + & = ?',
  ])('keeps the keyword %s intact', (keyword) => {
    const search = formatUrlView({
      kind: 'search',
      keyword,
      advanced: false,
      page: 1,
      queryLanguages: ['chs'],
      displayLanguages: ['chs'],
    })
    expect(parseUrlView(search)).toMatchObject({ kind: 'search', keyword })
  })
})

describe('languages', () => {
  it('keeps the order of both groups, alphabetical or not', () => {
    expect(parseUrlView('?q=a&lang=ja,chs&show=ja,en,chs')).toMatchObject({
      queryLanguages: ['ja', 'chs'],
      displayLanguages: ['ja', 'en', 'chs'],
    })
  })

  it('drops unknown and repeated codes without reordering the rest', () => {
    expect(
      parseUrlView('?q=a&lang=xx,ja,chs,ja&show=en,zz,en,tc'),
    ).toMatchObject({
      queryLanguages: ['ja', 'chs'],
      displayLanguages: ['en', 'tc'],
    })
  })

  it('treats a group that is empty once cleaned as absent', () => {
    const view = parseUrlView('?q=a&lang=xx&show=')
    expect(view).toMatchObject({ kind: 'search' })
    expect(view).not.toHaveProperty('queryLanguages', expect.anything())
    expect(view).not.toHaveProperty('displayLanguages', expect.anything())
    expect(resolveUrlView(view, fallback)).toMatchObject(fallback)
  })

  it('fills only the missing group from the fallback', () => {
    expect(
      resolveUrlView(parseUrlView('?q=a&show=en'), fallback),
    ).toMatchObject({ queryLanguages: ['chs'], displayLanguages: ['en'] })
  })
})

describe('page', () => {
  it.each(['0', '-1', 'x', '1.5', ''])('reads %s as page 1', (page) => {
    expect(parseUrlView(`?q=a&page=${page}`)).toMatchObject({ page: 1 })
  })

  it('does not write page 1', () => {
    expect(roundTrip('?q=a&page=1')).toBe('?q=a&lang=chs&show=chs,tc,en,ja')
  })
})

describe('mode', () => {
  it.each(['simple', 'ADVANCED', ''])('reads %s as plain mode', (mode) => {
    expect(parseUrlView(`?q=a&mode=${mode}`)).toMatchObject({
      advanced: false,
    })
  })
})

describe('which view', () => {
  it('opens the context view when sheet and at are valid, ignoring q', () => {
    expect(parseUrlView('?q=a&sheet=Item&at=0')).toEqual({
      kind: 'context',
      sheet: 'Item',
      index: 0,
      displayLanguages: undefined,
    })
  })

  it.each(['-1', 'x', ''])('falls back to the search when at is %s', (at) => {
    expect(parseUrlView(`?q=a&sheet=Item&at=${at}`)).toMatchObject({
      kind: 'search',
      keyword: 'a',
    })
  })

  it('is home without parameters, or with a blank keyword', () => {
    expect(parseUrlView('')).toEqual({ kind: 'home' })
    expect(parseUrlView('?q=%20%20')).toEqual({ kind: 'home' })
    expect(formatUrlView({ kind: 'home' })).toBe('')
  })
})

describe('queries', () => {
  it('turns a search view into the query it describes and back', () => {
    const view = resolveUrlView(
      parseUrlView('?q=a&lang=ja&show=en&mode=advanced&page=3'),
      fallback,
    )
    const query = viewToQuery(view, 20)
    expect(query?.keyword).toEqual({
      keyword: 'a',
      page: 3,
      pageSize: 20,
      languages: ['ja'],
      displayLanguages: ['en'],
      advanced: true,
    })
    if (query?.keyword) {
      expect(searchView(query.keyword, ['en'])).toEqual(view)
    }
  })

  it('opens the context window around the row, clamped at the first', () => {
    const query = viewToQuery(
      { kind: 'context', sheet: 'Item', index: 5, displayLanguages: ['en'] },
      20,
    )
    expect(query?.file).toEqual({
      sheet: 'Item',
      indexLower: 0,
      indexHigher: 25,
      displayLanguages: ['en'],
    })
  })
})
