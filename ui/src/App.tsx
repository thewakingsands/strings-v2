import { Spinner } from '@blueprintjs/core'
import styled from '@emotion/styled'
import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { useDebouncedCallback } from 'use-debounce'
import { DisplayLanguageBar } from './components/DisplayLanguageBar'
import { Footer } from './components/Footer'
import { MainContainer } from './components/MainContainer'
import { Pager } from './components/Pager'
import { ResultCount } from './components/ResultCount'
import { ResultTableHeader } from './components/ResultTableHeader'
import { SearchBar } from './components/SearchBar'
import { SearchError } from './components/SearchError'
import { SearchResult } from './components/SearchResult'
import { StaleResults } from './components/StaleResults'
import { TopNav } from './components/TopNav'
import type { StringItem } from './search/interface'
import type { IKeywordProps } from './search/linesByKeyword'
import {
  formatUrlView,
  parseUrlView,
  type ResolvedUrlView,
  resolveUrlView,
  searchView,
  type UrlView,
  viewToQuery,
} from './search/urlState'
import { useLanguageProbe } from './search/useLanguageProbe'
import { type ISearchQuery, useSearch } from './search/useSearch'
import {
  readInitialUrl,
  type UrlHistoryState,
  useUrlHistory,
} from './search/useUrlHistory'
import {
  readStoredDisplayLanguages,
  readStoredQueryLanguages,
} from './utils/language'
import { useIsMobile } from './utils/useIsMobile'
import {
  type LanguageChange,
  type LanguageSelection,
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

type HighlightItem = Pick<StringItem, 'sheet' | 'index'>

/** The search a context view goes back to, as its history entry recorded it. */
function previousQueryFrom(
  state: UrlHistoryState,
  fallback: LanguageSelection,
): ISearchQuery | null {
  if (!state.previousSearch) return null
  const view = parseUrlView(state.previousSearch)
  if (view.kind !== 'search') return null
  return viewToQuery(resolveUrlView(view, fallback), PAGE_SIZE) ?? null
}

function readInitialView() {
  const { view, state } = readInitialUrl()
  const stored: LanguageSelection = {
    queryLanguages: readStoredQueryLanguages(),
    displayLanguages: readStoredDisplayLanguages(),
  }
  return {
    view: resolveUrlView(view, stored),
    state,
    previousQuery:
      view.kind === 'context' ? previousQueryFrom(state, stored) : null,
  }
}

const contextView = (
  item: HighlightItem,
  displayLanguages: string[],
): ResolvedUrlView => ({
  kind: 'context',
  sheet: item.sheet,
  index: item.index,
  displayLanguages,
})

export default function App() {
  const [initial] = useState(readInitialView)
  const initialView = initial.view
  const [keywordInput, setKeywordInput] = useState(
    initialView.kind === 'search' ? initialView.keyword : '',
  )
  const [advanced, setAdvanced] = useState(
    initialView.kind === 'search' && initialView.advanced,
  )
  const [highlightItem, setHighlightItem] = useState<HighlightItem | null>(
    initialView.kind === 'context'
      ? { sheet: initialView.sheet, index: initialView.index }
      : null,
  )
  const [previousQuery, setPreviousQuery] = useState<ISearchQuery | null>(
    initial.previousQuery,
  )

  const search = useSearch(viewToQuery(initialView, PAGE_SIZE))
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

  const writeSearch = (
    keyword: IKeywordProps,
    displayLanguages: string[],
    mode: 'push' | 'replace',
    state?: UrlHistoryState,
  ) => {
    url.write(
      keyword.keyword.trim()
        ? searchView(keyword, displayLanguages)
        : { kind: 'home' },
      mode,
      state,
    )
  }

  const handleLanguageChange = ({
    queryLanguages,
    displayLanguages,
    reason,
  }: LanguageChange) => {
    if (reason === 'restore') return
    const file = search.query?.file
    if (!reasonNeedsRefetch(reason)) {
      // The columns are part of the link even though nothing has to be fetched for them.
      const submitted = search.query?.keyword
      if (submitted) writeSearch(submitted, displayLanguages, 'replace')
      else if (file && highlightItem) {
        url.write(contextView(highlightItem, displayLanguages), 'replace')
      }
      return
    }
    if (keywordInput) {
      const keyword: IKeywordProps = {
        keyword: keywordInput,
        page: 1,
        pageSize: PAGE_SIZE,
        languages: queryLanguages,
        displayLanguages,
        advanced,
      }
      // A pending search still carries the languages from before this change.
      debouncedSetSearch.cancel()
      search.setSearch({ keyword })
      writeSearch(keyword, displayLanguages, 'replace')
    } else if (file) {
      search.setSearch({
        file: { ...file, displayLanguages },
      })
      if (highlightItem) {
        url.write(contextView(highlightItem, displayLanguages), 'replace')
      }
    }
  }

  const languages = useLanguageSelection({
    onChange: handleLanguageChange,
    initial: initialView.kind === 'home' ? undefined : initialView,
  })
  const { queryLanguages, displayLanguages } = languages

  const debouncedSetSearch = useDebouncedCallback((q: ISearchQuery) => {
    setHighlightItem(null)
    search.setSearch(q)
    // Typing refines the one search rather than starting another, so it replaces the entry.
    if (q.keyword) {
      writeSearch(
        q.keyword,
        q.keyword.displayLanguages ?? displayLanguages,
        'replace',
        {},
      )
    }
  }, 400)

  const handlePop = (view: UrlView, state: UrlHistoryState) => {
    // Otherwise a search typed before going back fires afterwards and writes itself over it.
    debouncedSetSearch.cancel()
    const current = { queryLanguages, displayLanguages }
    const resolved = resolveUrlView(view, current)
    if (view.kind !== 'home') languages.restore(view)
    setKeywordInput(resolved.kind === 'search' ? resolved.keyword : '')
    if (resolved.kind === 'search') setAdvanced(resolved.advanced)
    setHighlightItem(
      resolved.kind === 'context'
        ? { sheet: resolved.sheet, index: resolved.index }
        : null,
    )
    setPreviousQuery(
      resolved.kind === 'context' ? previousQueryFrom(state, current) : null,
    )
    search.setSearch(viewToQuery(resolved, PAGE_SIZE))
  }

  const url = useUrlHistory(handlePop)

  // Rewrites a hand-made or partial link into the one the page now shows, so that copying the
  // address bar reproduces it exactly.
  useEffect(() => {
    url.write(initial.view, 'replace', initial.state)
  }, [url, initial])

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
      const keyword: IKeywordProps = {
        keyword: keywordInput,
        page: 1,
        pageSize: PAGE_SIZE,
        languages: queryLanguages,
        displayLanguages,
        advanced: newAdvanced,
      }
      search.setSearch({ keyword })
      writeSearch(keyword, displayLanguages, 'replace')
    }
  }

  const handleContextClick = (item: StringItem) => {
    debouncedSetSearch.cancel()
    const submitted = search.query?.keyword
    // The columns go back as they are now, not as they were when that search was sent.
    const back = submitted
      ? { ...submitted, displayLanguages }
      : previousQuery?.keyword
    if (submitted) setPreviousQuery({ keyword: back })
    setKeywordInput('')

    setHighlightItem(item)

    search.setSearch(
      viewToQuery(contextView(item, displayLanguages), PAGE_SIZE),
    )
    url.write(contextView(item, displayLanguages), 'push', {
      previousSearch: back
        ? formatUrlView(
            searchView(back, back.displayLanguages ?? displayLanguages),
          )
        : undefined,
    })
  }

  const handleBackClick = () => {
    const keyword = previousQuery?.keyword
    if (keyword) {
      search.setSearch(previousQuery)
      setPreviousQuery(null)
      setKeywordInput(keyword.keyword)
      setAdvanced(keyword.advanced ?? false)
      languages.restore({
        queryLanguages: keyword.languages.length
          ? keyword.languages
          : undefined,
        displayLanguages: keyword.displayLanguages,
      })
      // A view change like entering the context was, so the browser's back returns to it.
      writeSearch(
        keyword,
        keyword.displayLanguages ?? displayLanguages,
        'push',
        {},
      )
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
  // While a page loads this is still the count the last answer gave, which is what keeps the
  // pager on screen through the turn.
  const total = Math.ceil(search.result.total / PAGE_SIZE)
  const showPager = page > 0 && total > 0
  const showStale = search.isLoading && search.result.items.length > 0
  // A plain search matches any one of its words, so its total grows as the query gets more
  // specific. Only an advanced query's total counts what was asked for (STR-11).
  const showCount =
    !!search.resultQuery?.keyword?.advanced && search.result.total > 0

  // The rows stay put while the next page loads, so a turn from the bottom pager would otherwise
  // land at the bottom of the new page.
  const resultsRef = useRef<HTMLDivElement>(null)
  const pageTurned = useRef(false)
  useLayoutEffect(() => {
    if (search.isLoading || !pageTurned.current) return
    pageTurned.current = false
    const results = resultsRef.current
    if (!results) return
    const headerTop =
      Number.parseFloat(
        getComputedStyle(document.documentElement).getPropertyValue(
          '--results-header-top',
        ),
      ) || NAVBAR_HEIGHT
    const offset = results.getBoundingClientRect().top - headerTop
    if (offset < 0) window.scrollBy(0, offset)
  }, [search.isLoading])

  const handlePageChange = (page: number) => {
    pageTurned.current = true
    search.setPage(page)
    const submitted = search.query?.keyword
    if (submitted) writeSearch({ ...submitted, page }, displayLanguages, 'push')
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
          <ResultsSection ref={resultsRef}>
            {showCount && (
              <ResultCount total={search.result.total} busy={showStale} />
            )}
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
            {search.isLoading && !showStale ? (
              <Loading />
            ) : search.error ? (
              <SearchError error={search.error} />
            ) : (
              <StaleResults busy={showStale}>
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
              </StaleResults>
            )}
          </ResultsSection>
          {pager}
          <Footer />
        </MainContainer>
      </FillBodySection>
    </>
  )
}
