import { Button, InputGroup } from '@blueprintjs/core'
import type { CSSProperties } from 'react'

export interface ISearchFieldProps {
  keyword: string
  onKeywordChange: (keyword: string) => void
  className?: string
  style?: CSSProperties
}

export function SearchField(props: ISearchFieldProps) {
  return (
    <InputGroup
      className={props.className}
      large
      value={props.keyword}
      onChange={(e) => props.onKeywordChange(e.target.value)}
      leftIcon="search"
      placeholder="搜索"
      autoFocus
      rightElement={
        props.keyword ? (
          <Button
            icon="small-cross"
            minimal
            onClick={() => props.onKeywordChange('')}
          />
        ) : undefined
      }
    />
  )
}
