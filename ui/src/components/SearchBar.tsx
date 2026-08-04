import { Button, MenuItem, Tag } from '@blueprintjs/core'
import { MultiSelect } from '@blueprintjs/select'
import { css } from '@emotion/react'
import styled from '@emotion/styled'
import {
  formatQueryLanguagesLabel,
  type LanguageOption,
  languageOptions,
} from '@/utils/language'
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
  flexWrap: 'wrap',
})

const SearchInputContainer = styled.div({
  flexGrow: 100,
  flexBasis: '200px',
})

// Both language pickers share this so they stay the same width as each other.
const LanguageSelectContainer = styled.div({
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
  queryLanguages: string[]
  onQueryLanguagesChange: (queryLanguages: string[]) => void
  displayLanguages: string[]
  onDisplayLanguagesChange: (displayLanguages: string[]) => void
}

interface ItemRendererProps {
  handleClick: React.MouseEventHandler<HTMLElement>
  modifiers: { active: boolean }
}

const renderLanguageItem = (
  selected: string[],
  { showRank }: { showRank?: boolean } = {},
) =>
  function renderItem(
    item: LanguageOption,
    { handleClick, modifiers }: ItemRendererProps,
  ) {
    const rank = selected.indexOf(item.value)
    return (
      <MenuItem
        key={item.value}
        onClick={handleClick}
        active={modifiers.active}
        selected={rank !== -1}
        // MenuItem closes the popover on click by default, which would end the
        // selection after every single pick.
        shouldDismissPopover={false}
        text={item.label}
        // Query languages carry a priority, so the menu shows each pick's rank.
        labelElement={
          showRank && rank !== -1 ? (
            <Tag minimal round>
              {rank + 1}
            </Tag>
          ) : undefined
        }
        roleStructure="listoption"
      />
    )
  }

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
    <LanguageSelectContainer>
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
        itemRenderer={renderLanguageItem(value)}
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
    </LanguageSelectContainer>
  )
}

function QueryLanguageSelect({
  value,
  onChange,
}: {
  value: string[]
  onChange: (value: string[]) => void
}) {
  // Order is the priority the server boosts by, so keep the click order instead of
  // normalising like DisplayLanguagesSelect does.
  const selectedItems = value.flatMap(
    (lang) => languageOptions.find((opt) => opt.value === lang) ?? [],
  )

  const remove = (lang: string) => {
    // The server requires at least one language and answers 400 without it.
    if (value.length < 2) return
    onChange(value.filter((selected) => selected !== lang))
  }

  return (
    <LanguageSelectContainer>
      <MultiSelect<LanguageOption>
        customTarget={(items) => (
          <Button
            size="large"
            tabIndex={0}
            text={formatQueryLanguagesLabel(items.map((item) => item.value))}
            css={fullWidth}
            endIcon="caret-down"
          />
        )}
        items={languageOptions}
        selectedItems={selectedItems}
        itemRenderer={renderLanguageItem(value, { showRank: true })}
        itemPredicate={filterLanguage}
        onItemSelect={(item: LanguageOption) => {
          if (value.includes(item.value)) {
            remove(item.value)
          } else {
            // Appending keeps earlier picks at the higher priority.
            onChange([...value, item.value])
          }
        }}
        tagRenderer={(item: LanguageOption) => item.label}
        onRemove={(item: LanguageOption) => remove(item.value)}
        popoverProps={{ placement: 'bottom-start' }}
        placeholder="选择查询语言"
      />
    </LanguageSelectContainer>
  )
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
      <SearchContainer>
        {!previousQuery && (
          <>
            <QueryLanguageSelect
              value={props.queryLanguages}
              onChange={props.onQueryLanguagesChange}
            />
            <SearchInputContainer>
              <SearchField {...props} />
            </SearchInputContainer>
          </>
        )}
        <DisplayLanguagesSelect
          value={props.displayLanguages}
          onChange={props.onDisplayLanguagesChange}
        />
      </SearchContainer>
    </Container>
  )
}
