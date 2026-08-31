import { Spinner } from '@blueprintjs/core'
import styled from '@emotion/styled'
import { useState } from 'react'
import { useDebouncedCallback } from 'use-debounce'
import { Footer } from './components/Footer'
import { MainContainer } from './components/MainContainer'
import { Pager } from './components/Pager'
import { SearchBar } from './components/SearchBar'
import { SearchError } from './components/SearchError'
import { SearchResult } from './components/SearchResult'
import { TopNav } from './components/TopNav'
import type { StringItem } from './search/interface'
import { type ISearchQuery, useSearch } from './search/useSearch'
import {
  type LanguageChange,
  reasonNeedsRefetch,
  useLanguageSelection,
} from './utils/useLanguageSelection'

const MarginedDiv = styled.div({
  marginBottom: 12,
})

const FillBodySection = styled.section({
  minHeight: '100%',
})

const Loading = styled(Spinner)({
  minHeight: 200,
})

const StickyContainer = styled.div({
  position: 'sticky',
  top: 5,
  zIndex: 20,
})

const PAGE_SIZE = 20

export default function App() {
  const [keywordInput, setKeywordInput] = useState('')
  const [advanced, setAdvanced] = useState(false)
  const [highlightItem, setHighlightItem] = useState<StringItem | null>(null)
  const [previousQuery, setPreviousQuery] = useState<ISearchQuery | null>(null)

  const search = useSearch(undefined)

  const handleLanguageChange = ({
    queryLanguages,
    displayLanguages,
    reason,
  }: LanguageChange) => {
    if (!reasonNeedsRefetch(reason)) return
    if (keywordInput) {
      search.setSearch({
        keyword: {
          keyword: keywordInput,
          page: 1,
          pageSize: PAGE_SIZE,
          languages: queryLanguages,
          displayLanguages,
          advanced,
        },
      })
    }
    if (search.query?.file) {
      search.setSearch({
        file: { ...search.query.file, displayLanguages },
      })
    }
  }

  const languages = useLanguageSelection({ onChange: handleLanguageChange })
  const { queryLanguages, displayLanguages } = languages

  const debouncedSetSearch = useDebouncedCallback((q: ISearchQuery) => {
    setHighlightItem(null)
    search.setSearch(q)
  }, 400)

  const handleKeywordInputUpdate = (keyword: string) => {
    setKeywordInput(keyword)
    setPreviousQuery(null)
    const query: ISearchQuery = {
      keyword: {
        keyword,
        page: 1,
        pageSize: PAGE_SIZE,
        languages: queryLanguages,
        displayLanguages,
        advanced,
      },
    }
    debouncedSetSearch(query as ISearchQuery)
  }

  const handleAdvancedChange = (newAdvanced: boolean) => {
    setAdvanced(newAdvanced)
    // The same text reads differently in the other mode, so search it again.
    if (keywordInput) {
      search.setSearch({
        keyword: {
          keyword: keywordInput,
          page: 1,
          pageSize: PAGE_SIZE,
          languages: queryLanguages,
          displayLanguages,
          advanced: newAdvanced,
        },
      })
    }
  }

  const handleContextClick = (item: StringItem) => {
    if (keywordInput) {
      setPreviousQuery(search.query || null)
      setKeywordInput('')
    }

    setHighlightItem(item)

    const index = item.index
    search.setSearch({
      file: {
        sheet: item.sheet,
        indexLower: Math.max(0, index - 20),
        indexHigher: index + 20,
        displayLanguages,
      },
    })
  }

  const handleBackClick = () => {
    if (previousQuery) {
      search.setSearch(previousQuery)
      setPreviousQuery(null)
      setKeywordInput(previousQuery.keyword?.keyword || '')
      setAdvanced(previousQuery.keyword?.advanced ?? false)
      languages.restore({
        queryLanguages: previousQuery.keyword?.languages?.length
          ? previousQuery.keyword.languages
          : undefined,
        displayLanguages: previousQuery.keyword?.displayLanguages,
      })
    }
  }

  const page = search.query?.keyword?.page || 0
  const total = Math.ceil(search.result.total / PAGE_SIZE)
  const showPager = !search.isLoading && page > 0 && total > 0

  const handlePageChange = (page: number) => {
    search.setPage(page)
  }

  const pager = showPager && (
    <MarginedDiv>
      <Pager current={page} total={total} onPageChange={handlePageChange} />
    </MarginedDiv>
  )

  return (
    <>
      <TopNav
        displayLanguages={displayLanguages}
        onToggleDisplayLanguage={languages.toggleDisplayLanguage}
      />
      <FillBodySection>
        <MainContainer>
          <StickyContainer>
            <MarginedDiv>
              <SearchBar
                previousQuery={previousQuery || undefined}
                keyword={keywordInput}
                onKeywordChange={handleKeywordInputUpdate}
                advanced={advanced}
                onAdvancedChange={handleAdvancedChange}
                onBackClicked={handleBackClick}
                queryLanguages={queryLanguages}
                onToggleQueryLanguage={languages.toggleQueryLanguage}
                onReorderQueryLanguages={languages.reorderQueryLanguages}
              />
            </MarginedDiv>
          </StickyContainer>
          {pager}
          <MarginedDiv>
            {search.isLoading ? (
              <Loading />
            ) : search.error ? (
              <SearchError error={search.error} />
            ) : search.result ? (
              <SearchResult
                displayLanguages={displayLanguages}
                onHideLanguage={languages.hideLanguage}
                onToggleLanguage={languages.toggleDisplayLanguage}
                onReorderDisplayLanguages={languages.reorderDisplayLanguages}
                keyword={search.query?.keyword?.keyword || ''}
                advanced={search.query?.keyword?.advanced}
                items={search.result.items}
                onContextButtonClick={handleContextClick}
                highlightItem={highlightItem || undefined}
              />
            ) : null}
          </MarginedDiv>
          {pager}
          <Footer />
        </MainContainer>
      </FillBodySection>
    </>
  )
}
