import { describe, expect, it } from 'vitest'
import { pagerPages } from './pagerPages'

describe('pagerPages', () => {
  it('offers the first, the last and the siblings of a middle page', () => {
    expect(pagerPages(10, 22, 3)).toEqual([1, 7, 8, 9, 10, 11, 12, 13, 22])
    expect(pagerPages(10, 22, 2)).toEqual([1, 8, 9, 10, 11, 12, 22])
    expect(pagerPages(10, 22, 1)).toEqual([1, 9, 10, 11, 22])
  })

  it('stays inside the pages there are', () => {
    expect(pagerPages(1, 22, 3)).toEqual([1, 2, 3, 4, 22])
    expect(pagerPages(22, 22, 1)).toEqual([1, 21, 22])
  })

  it('lists a page once when the ranges meet', () => {
    expect(pagerPages(3, 5, 3)).toEqual([1, 2, 3, 4, 5])
    expect(pagerPages(1, 1, 3)).toEqual([1])
  })
})
