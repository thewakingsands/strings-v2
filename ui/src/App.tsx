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
  defaultDisplayLanguages,
  defaultQueryLanguages,
  sortToDisplayOrder,
} from './utils/language'

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

export default function App() {
  const [keywordInput, setKeywordInput] = useState('')
  const [queryLanguages, setQueryLanguages] = useState<string[]>([
    ...defaultQueryLanguages,
  ])
  const [displayLanguages, setDisplayLanguages] = useState<string[]>([
    ...defaultDisplayLanguages,
  ])
  const [highlightItem, setHighlightItem] = useState<StringItem | null>(null)
  const [previousQuery, setPreviousQuery] = useState<ISearchQuery | null>(null)

  const search = useSearch(undefined)

  const debouncedSetSearch = useDebouncedCallback((q: ISearchQuery) => {
    setHighlightItem(null)
    search.setSearch(q)
  }, 400)

  const PAGE_SIZE = 20

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
      },
    }
    debouncedSetSearch(query as ISearchQuery)
  }

  const handleQueryLanguagesChange = (newQueryLanguages: string[]) => {
    setQueryLanguages(newQueryLanguages)

    // A searched language that is not displayed comes back without its highlight,
    // so make sure every one of them has a column. Dropping a query language leaves
    // the column alone: removing it would take away something asked to be shown.
    const missing = newQueryLanguages.filter(
      (lang) => !displayLanguages.includes(lang),
    )
    let newDisplayLanguages = displayLanguages
    if (missing.length) {
      newDisplayLanguages = sortToDisplayOrder([
        ...displayLanguages,
        ...missing,
      ])
      setDisplayLanguages(newDisplayLanguages)
      // Keep an open file view in sync with the columns that just appeared.
      if (search.query?.file) {
        search.setSearch({
          file: {
            ...search.query.file,
            displayLanguages: newDisplayLanguages,
          },
        })
      }
    }

    // Trigger new search if there's a keyword
    if (keywordInput) {
      const query: ISearchQuery = {
        keyword: {
          keyword: keywordInput,
          page: 1,
          pageSize: PAGE_SIZE,
          languages: newQueryLanguages,
          displayLanguages: newDisplayLanguages,
        },
      }
      search.setSearch(query)
    }
  }

  const handleDisplayLanguagesChange = (newDisplayLanguages: string[]) => {
    setDisplayLanguages(newDisplayLanguages)
    // Trigger new search if there's a keyword
    if (keywordInput) {
      const query: ISearchQuery = {
        keyword: {
          keyword: keywordInput,
          page: 1,
          pageSize: PAGE_SIZE,
          languages: queryLanguages,
          displayLanguages: newDisplayLanguages,
        },
      }
      search.setSearch(query)
    }
    // Also update file view if active
    if (search.query?.file) {
      search.setSearch({
        file: {
          ...search.query.file,
          displayLanguages: newDisplayLanguages,
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
      if (previousQuery.keyword?.languages?.length) {
        setQueryLanguages(previousQuery.keyword.languages)
      }
      if (previousQuery.keyword?.displayLanguages) {
        setDisplayLanguages(previousQuery.keyword.displayLanguages)
      }
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
      <TopNav />
      <FillBodySection>
        <MainContainer>
          <StickyContainer>
            <MarginedDiv>
              <SearchBar
                previousQuery={previousQuery || undefined}
                keyword={keywordInput}
                onKeywordChange={handleKeywordInputUpdate}
                onBackClicked={handleBackClick}
                queryLanguages={queryLanguages}
                onQueryLanguagesChange={handleQueryLanguagesChange}
                displayLanguages={displayLanguages}
                onDisplayLanguagesChange={handleDisplayLanguagesChange}
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
                keyword={search.query?.keyword?.keyword || ''}
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
