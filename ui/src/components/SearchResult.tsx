import { NoResult } from './NoResult'
import { type IResultTableProps, ResultTable } from './ResultTable'

export function SearchResult(props: IResultTableProps) {
  const resultCount = props.items.length
  if (resultCount < 1) {
    return <NoResult keyword={props.keyword} />
  } else {
    return <ResultTable {...props} />
  }
}
