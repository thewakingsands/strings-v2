import { useState } from 'react'
import {
  readStoredDisplayLanguages,
  readStoredQueryLanguages,
  writeStoredDisplayLanguages,
  writeStoredQueryLanguages,
} from './language'

export type LanguageChangeReason =
  | 'query'
  | 'query-order'
  | 'display'
  | 'column-order'
  | 'restore'

export interface LanguageSelection {
  queryLanguages: string[]
  displayLanguages: string[]
}

export interface LanguageChange extends LanguageSelection {
  reason: LanguageChangeReason
}

/**
 * Column order only decides how the columns are painted — the values come back keyed by language,
 * so nothing has to be fetched again. A restore replays a query the caller has already issued.
 */
export function reasonNeedsRefetch(reason: LanguageChangeReason): boolean {
  return reason !== 'column-order' && reason !== 'restore'
}

/**
 * Which stored group a change writes. A restore replays a selection that came from a link or from
 * history, not from the visitor's hands, so it writes neither.
 */
export function reasonPersists(
  reason: LanguageChangeReason,
): 'query' | 'display' | null {
  switch (reason) {
    case 'query':
    case 'query-order':
      return 'query'
    case 'display':
    case 'column-order':
      return 'display'
    case 'restore':
      return null
  }
}

/** The transitions below answer null when a rule refuses, so nothing is dispatched for a no-op. */

export function toggleQuery(current: string[], code: string): string[] | null {
  if (!current.includes(code)) return [...current, code]
  // The server answers 400 without a lang parameter, so the last one cannot go.
  if (current.length < 2) return null
  return current.filter((lang) => lang !== code)
}

export function replaceQuery(
  current: string[],
  codes: string[],
): string[] | null {
  if (codes.length === 0) return null
  if (
    codes.length === current.length &&
    codes.every((code, i) => code === current[i])
  ) {
    return null
  }
  return [...codes]
}

export function showDisplay(current: string[], code: string): string[] | null {
  if (current.includes(code)) return null
  // Appended rather than sorted: the order is the column order the user dragged.
  return [...current, code]
}

export function hideDisplay(current: string[], code: string): string[] | null {
  if (!current.includes(code)) return null
  // Emptying the table from the UI has no use; a stored empty list is still restored as is.
  if (current.length < 2) return null
  return current.filter((lang) => lang !== code)
}

export function reorder(
  current: string[],
  from: number,
  to: number,
): string[] | null {
  if (from === to) return null
  if (from < 0 || to < 0 || from >= current.length || to >= current.length) {
    return null
  }
  const next = [...current]
  const [moved] = next.splice(from, 1)
  next.splice(to, 0, moved)
  return next
}

export interface UseLanguageSelection extends LanguageSelection {
  toggleQueryLanguage: (code: string) => void
  reorderQueryLanguages: (from: number, to: number) => void
  setQueryLanguages: (codes: string[]) => void
  showLanguage: (code: string) => void
  hideLanguage: (code: string) => void
  /** What the checkable menu needs: one entry point for both directions. */
  toggleDisplayLanguage: (code: string) => void
  reorderDisplayLanguages: (from: number, to: number) => void
  restore: (next: Partial<LanguageSelection>) => void
}

/**
 * Owns both language groups and every rule about them, so that persistence and the "does this
 * need a new request" decision each live in one place instead of in every handler.
 */
export function useLanguageSelection({
  onChange,
  initial,
}: {
  onChange: (change: LanguageChange) => void
  /** Groups to open with instead of the stored ones, as a link carries them. */
  initial?: Partial<LanguageSelection>
}): UseLanguageSelection {
  const [queryLanguages, setQueryLanguages] = useState<string[]>(
    () => initial?.queryLanguages ?? readStoredQueryLanguages(),
  )
  const [displayLanguages, setDisplayLanguages] = useState<string[]>(
    () => initial?.displayLanguages ?? readStoredDisplayLanguages(),
  )

  // Writing only from the visitor's own changes, and only the group they changed, keeps anyone
  // who has not touched a picker on its defaults — so a later change to those defaults still
  // reaches them — and keeps a link's languages out of storage until the visitor adopts them.
  const applyQuery = (
    next: string[] | null,
    reason: LanguageChangeReason,
  ): void => {
    if (!next) return
    setQueryLanguages(next)
    if (reasonPersists(reason) === 'query') writeStoredQueryLanguages(next)
    onChange({ queryLanguages: next, displayLanguages, reason })
  }

  const applyDisplay = (
    next: string[] | null,
    reason: LanguageChangeReason,
  ): void => {
    if (!next) return
    setDisplayLanguages(next)
    if (reasonPersists(reason) === 'display') writeStoredDisplayLanguages(next)
    onChange({ queryLanguages, displayLanguages: next, reason })
  }

  return {
    queryLanguages,
    displayLanguages,
    toggleQueryLanguage: (code) =>
      applyQuery(toggleQuery(queryLanguages, code), 'query'),
    reorderQueryLanguages: (from, to) =>
      applyQuery(reorder(queryLanguages, from, to), 'query-order'),
    setQueryLanguages: (codes) =>
      applyQuery(replaceQuery(queryLanguages, codes), 'query'),
    showLanguage: (code) =>
      applyDisplay(showDisplay(displayLanguages, code), 'display'),
    hideLanguage: (code) =>
      applyDisplay(hideDisplay(displayLanguages, code), 'display'),
    toggleDisplayLanguage: (code) =>
      applyDisplay(
        displayLanguages.includes(code)
          ? hideDisplay(displayLanguages, code)
          : showDisplay(displayLanguages, code),
        'display',
      ),
    reorderDisplayLanguages: (from, to) =>
      applyDisplay(reorder(displayLanguages, from, to), 'column-order'),
    restore: (next) => {
      const query = next.queryLanguages ?? queryLanguages
      const display = next.displayLanguages ?? displayLanguages
      setQueryLanguages(query)
      setDisplayLanguages(display)
      onChange({
        queryLanguages: query,
        displayLanguages: display,
        reason: 'restore',
      })
    },
  }
}
