import { NoResult } from './NoResult'
import { type IResultTableProps, ResultTable } from './ResultTable'

export interface ISearchResultProps extends IResultTableProps {
  /** Whether keyword is a query string, whose syntax is no words to highlight. */
  advanced?: boolean
}

export function SearchResult({ advanced, ...props }: ISearchResultProps) {
  const resultCount = props.items.length
  if (resultCount < 1) {
    return <NoResult keyword={props.keyword} />
  } else {
    // The server's marks still come through; only the browser's own guesses from the
    // query text are switched off.
    return <ResultTable {...props} keyword={advanced ? '' : props.keyword} />
  }
}
