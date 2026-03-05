import { Button } from '@blueprintjs/core'
import { MultiSelect, Select } from '@blueprintjs/select'
import styled from '@emotion/styled'
import { type LanguageOption, languageOptions } from '@/utils/language'
import type { ISearchQuery } from '../search/useSearch'
import { type ISearchFieldProps, SearchField } from './SearchField'
import { css } from '@emotion/react'

const Container = styled.div({
  display: 'flex',
  gap: '12px',
  alignItems: 'flex-start',
})

const SearchContainer = styled.div({
  display: 'flex',
  gap: '8px',
  flex: 1,
  flexWrap: 'wrap',
})

const SearchInputContainer = styled.div({
  flexGrow: 100,
  flexBasis: '200px',
})

const SelectContainer = styled.div({
  flexGrow: 1,
  flexShrink: 0,
  flexBasis: '120px',
})

const MultiSelectContainer = styled.div({
  flexGrow: 1,
  flexShrink: 0,
  flexBasis: '150px',
})

const fullWidth = css({
  width: '100%',
})

export interface ISearchBarProps extends ISearchFieldProps {
  previousQuery?: ISearchQuery
  onBackClicked?: () => void
  language: string
  onLanguageChange: (language: string) => void
  displayLanguages: string[]
  onDisplayLanguagesChange: (displayLanguages: string[]) => void
}

const renderLanguageItem = (
  item: LanguageOption,
  {
    handleClick,
    modifiers,
  }: {
    handleClick: React.MouseEventHandler<HTMLElement>
    modifiers: { active: boolean }
  },
) => (
  <div
    key={item.value}
    onClick={handleClick}
    style={{
      padding: '8px',
      cursor: 'pointer',
      backgroundColor: modifiers.active ? '#e5e5e5' : 'transparent',
    }}
  >
    {item.label}
  </div>
)

const filterLanguage = (query: string, item: LanguageOption): boolean => {
  const normalizedQuery = query.toLowerCase()
  return (
    item.label.toLowerCase().includes(normalizedQuery) ||
    item.value.toLowerCase().includes(normalizedQuery)
  )
}

function DisplayLanguagesSelect({
  value,
  onChange,
}: {
  value: string[]
  onChange: (value: string[]) => void
}) {
  return (
    <MultiSelectContainer>
      <MultiSelect<LanguageOption>
        customTarget={(items) => (
          <Button
            size="large"
            tabIndex={0}
            text={`显示语言 (${items.length})`}
            css={fullWidth}
            endIcon="caret-down"
          />
        )}
        items={languageOptions}
        selectedItems={languageOptions.filter((opt) =>
          value.includes(opt.value),
        )}
        itemRenderer={renderLanguageItem}
        itemPredicate={filterLanguage}
        onItemSelect={(item: LanguageOption) => {
          if (!value.includes(item.value)) {
            const newSelection = languageOptions
              .map((opt) => opt.value)
              .filter((lang) => value.includes(lang) || lang === item.value)
            onChange(newSelection)
          }
        }}
        tagRenderer={(item: LanguageOption) => item.label}
        onRemove={(item: LanguageOption) => {
          onChange(value.filter((lang) => lang !== item.value))
        }}
        popoverProps={{ placement: 'bottom-end' }}
        placeholder="选择显示语言"
      />
    </MultiSelectContainer>
  )
}

function QueryLanguageSelect({
  value,
  onChange,
}: {
  value: string
  onChange: (value: string) => void
}) {
  return (
    <SelectContainer>
      <Select<LanguageOption>
        items={languageOptions}
        itemRenderer={renderLanguageItem}
        itemPredicate={filterLanguage}
        onItemSelect={(item: LanguageOption) => onChange(item.value)}
        filterable={false}
        popoverProps={{ placement: 'bottom-start' }}
      >
        <Button
          size="large"
          text={
            languageOptions.find((opt) => opt.value === value)?.label ||
            '选择查询语言'
          }
          css={fullWidth}
          endIcon="caret-down"
        />
      </Select>
    </SelectContainer>
  )
}

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
        <QueryLanguageSelect
          value={props.language}
          onChange={props.onLanguageChange}
        />
        <SearchInputContainer>
          <SearchField {...props} />
        </SearchInputContainer>
        <DisplayLanguagesSelect
          value={props.displayLanguages}
          onChange={props.onDisplayLanguagesChange}
        />
      </SearchContainer>
    </Container>
  )
}
