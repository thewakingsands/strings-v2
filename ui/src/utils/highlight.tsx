import type { ReactNode } from 'react'

export function highlightText(text: string, query: string): ReactNode {
  if (!query || !text) return text
  const lowerText = text.toLowerCase()
  const lowerQuery = query.toLowerCase()
  const index = lowerText.indexOf(lowerQuery)

  if (index === -1) return text

  const before = text.substring(0, index)
  const match = text.substring(index, index + query.length)
  const after = text.substring(index + query.length)

  return (
    <span>
      {before}
      <em>{match}</em>
      {after}
    </span>
  )
}
