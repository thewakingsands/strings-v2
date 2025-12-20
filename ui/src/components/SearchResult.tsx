import type { StringItem } from '../search/interface'
import { NoResult } from './NoResult'
import { ResultTable } from './ResultTable'

export interface ISearchResultProps {
  keyword: string
  items: StringItem[]
  highlightItem?: Pick<StringItem, 'sheet' | 'rowId'>
  onContextButtonClick?: (item: StringItem) => void
}

export function SearchResult(props: ISearchResultProps) {
  const resultCount = props.items.length
  if (resultCount < 1) {
    return <NoResult keyword={props.keyword} />
  } else {
    return <ResultTable {...props} />
  }
}
