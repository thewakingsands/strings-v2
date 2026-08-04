import type { ReactNode } from 'react'
import { resolveHighlightSegments } from './highlightSegments'

/**
 * Renders text with its highlighted runs wrapped in <em>.
 *
 * Set `escaped` for a server side highlight fragment; see resolveHighlightSegments.
 */
export function highlightText(
  text: string,
  query: string,
  escaped = false,
): ReactNode {
  return resolveHighlightSegments(text, query, escaped).map((segment, i) =>
    segment.marked ? (
      // Segments come out in document order, so the index is a stable key.
      // biome-ignore lint/suspicious/noArrayIndexKey: order is fixed by the text
      <em key={i}>{segment.text}</em>
    ) : (
      segment.text
    ),
  )
}
