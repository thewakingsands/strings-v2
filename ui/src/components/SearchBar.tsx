import { Button } from '@blueprintjs/core'
import styled from '@emotion/styled'
import { languageOptions } from '@/utils/language'
import type { ISearchQuery } from '../search/useSearch'
import { LanguageChips } from './LanguageChips'
import { type ISearchFieldProps, SearchField } from './SearchField'

const Container = styled.div({
  display: 'flex',
  gap: '12px',
  alignItems: 'flex-start',
})

const SearchColumn = styled.div({
  display: 'flex',
  flexDirection: 'column',
  gap: '8px',
  flex: 1,
  minWidth: 0,
})

export interface ISearchBarProps extends ISearchFieldProps {
  previousQuery?: ISearchQuery
  onBackClicked?: () => void
  queryLanguages: string[]
  onToggleQueryLanguage: (code: string) => void
  onReorderQueryLanguages: (from: number, to: number) => void
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
      {/* The context view has no keyword search, so neither the field nor the languages
          modifying it belong there. The display languages live on the result table and in
          the navbar, so that view needs no language control of its own. */}
      {!previousQuery && (
        <SearchColumn>
          <SearchField {...props} />
          <LanguageChips
            label="搜索"
            options={languageOptions}
            selected={props.queryLanguages}
            onToggle={props.onToggleQueryLanguage}
            onReorder={props.onReorderQueryLanguages}
            showRank
          />
        </SearchColumn>
      )}
    </Container>
  )
}
