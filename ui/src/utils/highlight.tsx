import type { ReactNode } from 'react'

const unescapeHtmlTags = (text: string) => {
  return text.replace(/&lt;/g, '<').replace(/&gt;/g, '>')
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

  if (index === -1) return [unescapeHtmlTags(text)]

  const before = text.substring(0, index)
  const match = text.substring(index, index + query.length)
  const after = text.substring(index + query.length)

  return [
    unescapeHtmlTags(before),
    <em key={`match-${lastIndex}-${index}`}>{match}</em>,
    unescapeHtmlTags(after),
  ]
}

const markRegex = /<mark>(.*?)<\/mark>/gi
export function highlightText(text: string, query: string): ReactNode {
  if (!text || !query) return text

  let pos = 0
  const elements: ReactNode[] = []
  let match: RegExpExecArray | null

  while ((match = markRegex.exec(text)) !== null) {
    // Text before <mark>
    if (match.index > pos) {
      const before = text.substring(pos, match.index)
      elements.push(...highlightTextWithoutMark(before, query, pos))
    }

    elements.push(<em key={`mark-${match.index}`}>{match[1]}</em>)
    pos = match.index + match[0].length
  }

  if (pos < text.length) {
    elements.push(...highlightTextWithoutMark(text.substring(pos), query, pos))
  }

  return elements
}
