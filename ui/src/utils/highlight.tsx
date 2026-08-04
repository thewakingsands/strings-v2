import type { ReactNode } from 'react'

const htmlEntities: Record<string, string> = {
  '&amp;': '&',
  '&lt;': '<',
  '&gt;': '>',
  '&#34;': '"',
  '&quot;': '"',
  '&#39;': "'",
}

// One pass over the text, so `&amp;lt;` decodes to `&lt;` rather than being decoded
// twice down to `<`.
const htmlEntityRegex = /&(?:amp|lt|gt|quot|#34|#39);/g

function unescapeHtmlEntities(text: string): string {
  return text.replace(
    htmlEntityRegex,
    (entity) => htmlEntities[entity] ?? entity,
  )
}

function identity(text: string): string {
  return text
}

function highlightTextWithoutMark(
  text: string,
  query: string,
  lastIndex: number,
): ReactNode[] {
  if (!query || !text) return [text]
  const lowerText = text.toLowerCase()
  const lowerQuery = query.toLowerCase()
  const index = lowerText.indexOf(lowerQuery)

  if (index === -1) return [text]

  const before = text.substring(0, index)
  const match = text.substring(index, index + query.length)
  const after = text.substring(index + query.length)

  return [before, <em key={`match-${lastIndex}-${index}`}>{match}</em>, after]
}

const markRegex = /<mark>(.*?)<\/mark>/gi

/**
 * Renders text with the server side <mark> spans, plus any remaining literal
 * occurrences of the query, turned into <em> elements.
 *
 * Set `escaped` for a server side highlight fragment, whose text the search index
 * HTML escaped. Complete values arrive raw, and decoding those would turn an entity
 * that the game text itself contains into the character it stands for.
 */
export function highlightText(
  text: string,
  query: string,
  escaped = false,
): ReactNode {
  const decode = escaped ? unescapeHtmlEntities : identity

  if (!text) return text
  if (!query) return decode(text)

  let pos = 0
  const elements: ReactNode[] = []
  let match: RegExpExecArray | null

  while ((match = markRegex.exec(text)) !== null) {
    // Text before <mark>
    if (match.index > pos) {
      const before = decode(text.substring(pos, match.index))
      elements.push(...highlightTextWithoutMark(before, query, pos))
    }

    elements.push(<em key={`mark-${match.index}`}>{decode(match[1])}</em>)
    pos = match.index + match[0].length
  }

  if (pos < text.length) {
    const rest = decode(text.substring(pos))
    elements.push(...highlightTextWithoutMark(rest, query, pos))
  }

  return elements
}
