export interface StringItem {
  sheet: string
  rowId: string
  values: Record<string, string>
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
