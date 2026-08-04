export interface StringItem {
  sheet: string
  rowId: string
  /** Complete, raw values, one entry per requested language. */
  values: Record<string, string>
  /**
   * Matching snippet per language that the search hit: HTML escaped, matches wrapped
   * in <mark>, shortened with an ellipsis. Absent for languages that did not match.
   */
  highlights?: Record<string, string>
  index: number
}

export interface SearchResult {
  items: StringItem[]
  total: number
}

export const emptySearchResult: SearchResult = Object.freeze({
  items: [],
  total: 0,
})
