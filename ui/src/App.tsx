import { Spinner } from '@blueprintjs/core'
import styled from '@emotion/styled'
import { useLayoutEffect, useRef, useState } from 'react'
import { useDebouncedCallback } from 'use-debounce'
import { DisplayLanguageBar } from './components/DisplayLanguageBar'
import { Footer } from './components/Footer'
import { MainContainer } from './components/MainContainer'
import { Pager } from './components/Pager'
import { ResultTableHeader } from './components/ResultTableHeader'
import { SearchBar } from './components/SearchBar'
import { SearchError } from './components/SearchError'
import { SearchResult } from './components/SearchResult'
import { TopNav } from './components/TopNav'
import type { StringItem } from './search/interface'
import { useLanguageProbe } from './search/useLanguageProbe'
import { type ISearchQuery, useSearch } from './search/useSearch'
import { useIsMobile } from './utils/useIsMobile'
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

// The field sticks over the navbar's own band, which saves a row of height on every scroll.
const STICKY_TOP = 5
const NAVBAR_HEIGHT = 50

const StickyContainer = styled.div({
  position: 'sticky',
  top: STICKY_TOP,
  zIndex: 20,
})

// The results header sticks within this, so it leaves along with the last result.
const ResultsSection = styled.div({
  marginBottom: 12,
})

const PAGE_SIZE = 20

export default function App() {
  const [keywordInput, setKeywordInput] = useState('')
  const [advanced, setAdvanced] = useState(false)
  const [highlightItem, setHighlightItem] = useState<StringItem | null>(null)
  const [previousQuery, setPreviousQuery] = useState<ISearchQuery | null>(null)

  const search = useSearch(undefined)
  const isMobile = useIsMobile()

  // The results header sticks just under the field, whose height changes with the back button
  // and with the fonts, so it is measured rather than written down.
  const stickyRef = useRef<HTMLDivElement>(null)
  useLayoutEffect(() => {
    const sticky = stickyRef.current
    if (!sticky) return
    const root = document.documentElement
    const observer = new ResizeObserver(() => {
      const bottom = STICKY_TOP + sticky.offsetHeight
      root.style.setProperty(
        '--results-header-top',
        `${Math.max(NAVBAR_HEIGHT, bottom)}px`,
      )
    })
    observer.observe(sticky)
    return () => observer.disconnect()
  }, [])

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

  const submitted = search.query?.keyword
  // Advanced clauses name their fields, which lang does not narrow, so counts per language
  // would mislead there.
  const probeEnabled =
    !!submitted?.keyword &&
    !submitted.advanced &&
    submitted.page === 1 &&
    !search.isLoading &&
    !search.error &&
    search.result.total === 0
  const probe = useLanguageProbe({
    keyword: submitted?.keyword ?? '',
    queryLanguages: submitted?.languages ?? [],
    enabled: probeEnabled,
  })

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
          <StickyContainer ref={stickyRef}>
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
          <ResultsSection>
            {/* Rendered in every state, landing page and empty result included: this is the
                only place the display languages can be changed. */}
            {isMobile ? (
              <DisplayLanguageBar
                displayLanguages={displayLanguages}
                onToggle={languages.toggleDisplayLanguage}
                onReorder={languages.reorderDisplayLanguages}
              />
            ) : (
              <ResultTableHeader
                displayLanguages={displayLanguages}
                onHideLanguage={languages.hideLanguage}
                onToggleLanguage={languages.toggleDisplayLanguage}
                onReorderDisplayLanguages={languages.reorderDisplayLanguages}
              />
            )}
            {search.isLoading ? (
              <Loading />
            ) : search.error ? (
              <SearchError error={search.error} />
            ) : search.result ? (
              <SearchResult
                displayLanguages={displayLanguages}
                keyword={search.query?.keyword?.keyword || ''}
                advanced={search.query?.keyword?.advanced}
                items={search.result.items}
                onContextButtonClick={handleContextClick}
                highlightItem={highlightItem || undefined}
                fallback={
                  probeEnabled && submitted
                    ? {
                        keyword: submitted.keyword,
                        searchedLanguages: submitted.languages,
                        loading: probe.loading,
                        ranked: probe.ranked,
                        onReplace: (lang) =>
                          languages.setQueryLanguages([lang]),
                        onAdd: languages.toggleQueryLanguage,
                      }
                    : undefined
                }
              />
            ) : null}
          </ResultsSection>
          {pager}
          <Footer />
        </MainContainer>
      </FillBodySection>
    </>
  )
}
