import { Button, HTMLSelect } from '@blueprintjs/core'
import styled from '@emotion/styled'
import { languageMap } from '@/utils/language'
import type { ISearchQuery } from '../search/useSearch'
import { type ISearchFieldProps, SearchField } from './SearchField'

const Container = styled.div({
  display: 'flex',
  gap: '12px',
  alignItems: 'flex-start',
})

const SearchContainer = styled.div({
  display: 'flex',
  gap: '8px',
  flex: 1,
})

const LanguageSelect = styled(HTMLSelect)({
  minWidth: '100px',
})

export interface ISearchBarProps extends ISearchFieldProps {
  previousQuery?: ISearchQuery
  onBackClicked?: () => void
  language: string
  onLanguageChange: (language: string) => void
}

const LANGUAGE_OPTIONS = Object.entries(languageMap).map(([value, label]) => ({
  value,
  label,
}))

export function SearchBar(props: ISearchBarProps) {
  const kw = props.previousQuery?.keyword?.keyword
  const text = kw ? `返回搜索"${kw}"` : undefined

  return (
    <Container>
      {props.previousQuery && (
        <Button
          onClick={() => props.onBackClicked?.()}
          text={text}
          size="large"
          icon="chevron-left"
          intent="primary"
        />
      )}
      <SearchContainer>
        <LanguageSelect
          value={props.language}
          onChange={(e) => props.onLanguageChange(e.target.value)}
          options={LANGUAGE_OPTIONS}
          large
        />
        <div style={{ flex: 1 }}>
          <SearchField {...props} />
        </div>
      </SearchContainer>
    </Container>
  )
}
