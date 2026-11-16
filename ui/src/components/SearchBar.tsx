import { Button } from '@blueprintjs/core'
import styled from '@emotion/styled'
import type { ISearchQuery } from '../search/useSearch'
import { type ISearchFieldProps, SearchField } from './SearchField'

const Container = styled.div({
  display: 'flex',
  gap: '12px',
  alignItems: 'flex-start',
})

const SearchColumn = styled.div({
  flex: 1,
  minWidth: 0,
})

export interface ISearchBarProps extends ISearchFieldProps {
  previousQuery?: ISearchQuery
  onBackClicked?: () => void
}

export function SearchBar(props: ISearchBarProps) {
  const { previousQuery } = props
  const kw = previousQuery?.keyword?.keyword
  const text = kw ? `返回搜索"${kw}"` : undefined

  return (
    <Container>
      {previousQuery && (
        <Button
          onClick={() => props.onBackClicked?.()}
          text={text}
          size="large"
          icon="chevron-left"
          intent="primary"
        />
      )}
      {/* The context view has no keyword search, so neither the field nor the search languages
          it carries belong there. The display languages sit on top of the results, which that
          view has too. */}
      {!previousQuery && (
        <SearchColumn>
          <SearchField {...props} />
        </SearchColumn>
      )}
    </Container>
  )
}
