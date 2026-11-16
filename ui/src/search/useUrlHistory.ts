import { useEffect, useRef } from 'react'
import {
  formatUrlView,
  parseUrlView,
  type ResolvedUrlView,
  type UrlView,
} from './urlState'

export interface UrlHistoryState {
  /** The query string of the search a context view was opened from. */
  previousSearch?: string
}

function readState(): UrlHistoryState {
  const state: unknown = history.state
  return state && typeof state === 'object' ? (state as UrlHistoryState) : {}
}

export function readInitialUrl(): { view: UrlView; state: UrlHistoryState } {
  return { view: parseUrlView(location.search), state: readState() }
}

function write(
  view: ResolvedUrlView,
  mode: 'push' | 'replace',
  state?: UrlHistoryState,
): void {
  const search = formatUrlView(view)
  if (mode === 'replace') {
    if (state === undefined && search === location.search) return
    // A replace keeps the entry's record unless told otherwise: changing the columns of a
    // context view must not lose the search it goes back to.
    history.replaceState(state ?? readState(), '', location.pathname + search)
  } else {
    history.pushState(state ?? {}, '', location.pathname + search)
  }
}

// One object for every render, so an effect can depend on it without running on each one.
const urlHistory = { write }

/**
 * Callers say whether a change gets its own history entry, rather than having an effect guess it
 * from the difference between two states: a page turn and a language change look alike there.
 */
export function useUrlHistory(
  onPop: (view: UrlView, state: UrlHistoryState) => void,
): typeof urlHistory {
  const onPopRef = useRef(onPop)
  onPopRef.current = onPop

  useEffect(() => {
    const listener = () =>
      onPopRef.current(parseUrlView(location.search), readState())
    window.addEventListener('popstate', listener)
    return () => window.removeEventListener('popstate', listener)
  }, [])

  return urlHistory
}
