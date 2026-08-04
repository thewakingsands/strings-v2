export interface HighlightSegment {
  text: string
  marked: boolean
}

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

export function unescapeHtmlEntities(text: string): string {
  return text.replace(
    htmlEntityRegex,
    (entity) => htmlEntities[entity] ?? entity,
  )
}

const markRegex = /<mark>(.*?)<\/mark>/gis
const wordRegex = /[\p{L}\p{N}]+/gu

interface Token {
  text: string
  isWord: boolean
  marked: boolean
}

/** Splits the server's <mark> spans out, decoding each piece when it is escaped. */
function parseMarks(text: string, decode: (s: string) => string): Token[] {
  const pieces: HighlightSegment[] = []
  let pos = 0

  for (const match of text.matchAll(markRegex)) {
    const start = match.index
    if (start > pos) {
      pieces.push({ text: decode(text.slice(pos, start)), marked: false })
    }
    pieces.push({ text: decode(match[1]), marked: true })
    pos = start + match[0].length
  }
  if (pos < text.length) {
    pieces.push({ text: decode(text.slice(pos)), marked: false })
  }

  return pieces.flatMap(tokenize)
}

/** Cuts a piece into words and the separators between them, keeping its marked flag. */
function tokenize(piece: HighlightSegment): Token[] {
  const tokens: Token[] = []
  let pos = 0

  for (const match of piece.text.matchAll(wordRegex)) {
    const start = match.index
    if (start > pos) {
      tokens.push({
        text: piece.text.slice(pos, start),
        isWord: false,
        marked: piece.marked,
      })
    }
    tokens.push({ text: match[0], isWord: true, marked: piece.marked })
    pos = start + match[0].length
  }
  if (pos < piece.text.length) {
    tokens.push({
      text: piece.text.slice(pos),
      isWord: false,
      marked: piece.marked,
    })
  }

  return tokens
}

function queryWords(query: string): string[] {
  return query.toLowerCase().match(wordRegex) ?? []
}

/**
 * Maps each query word to the words that sit right next to it in the query. Sets,
 * because a word may appear more than once with different neighbours.
 */
function adjacency(words: string[]) {
  const next = new Map<string, Set<string>>()
  const previous = new Map<string, Set<string>>()

  for (let i = 0; i + 1 < words.length; i++) {
    const [left, right] = [words[i], words[i + 1]]
    if (!next.has(left)) next.set(left, new Set())
    next.get(left)?.add(right)
    if (!previous.has(right)) previous.set(right, new Set())
    previous.get(right)?.add(left)
  }

  return { next, previous }
}

/**
 * Marks the words the search engine could not: a stop word carries no term location,
 * so the server never highlights it even though the user typed it.
 *
 * A word is taken in when the query says it belongs right there — its neighbour in the
 * text is marked, and the query has the two of them adjacent in the same order. That
 * double condition is what keeps an unrelated "to" elsewhere in the line untouched.
 * Repeating until nothing changes lets a run of stop words be absorbed one by one, as
 * in "out of the box".
 */
function expandMarks(tokens: Token[], query: string): void {
  const { next, previous } = adjacency(queryWords(query))
  if (next.size === 0) return

  const wordAt = tokens.flatMap((token, i) => (token.isWord ? i : []))

  let changed = true
  while (changed) {
    changed = false

    for (let k = 0; k < wordAt.length; k++) {
      const token = tokens[wordAt[k]]
      if (token.marked) continue

      const word = token.text.toLowerCase()
      const left = k > 0 ? tokens[wordAt[k - 1]] : undefined
      const right = k + 1 < wordAt.length ? tokens[wordAt[k + 1]] : undefined

      const afterLeft =
        left?.marked && next.get(left.text.toLowerCase())?.has(word)
      const beforeRight =
        right?.marked && previous.get(right.text.toLowerCase())?.has(word)

      if (afterLeft || beforeRight) {
        token.marked = true
        changed = true
      }
    }
  }

  joinNeighbouringMarks(tokens, wordAt, next)
}

/**
 * Pulls what sits between two marked words into the highlight, so a run reads as one
 * continuous span instead of several boxes with gaps between them.
 *
 * Only when the query has those two words adjacent as well. Two hits that happen to
 * land side by side in the text without being neighbours in the query stay separate,
 * otherwise the highlight would suggest a phrase that is not there: searching
 * "can hardly afford" over a line reading "we can afford" would join "can afford" and
 * hide that the middle word is missing.
 */
function joinNeighbouringMarks(
  tokens: Token[],
  wordAt: number[],
  next: Map<string, Set<string>>,
): void {
  for (let k = 0; k + 1 < wordAt.length; k++) {
    const [left, right] = [tokens[wordAt[k]], tokens[wordAt[k + 1]]]
    if (!left.marked || !right.marked) continue
    if (!next.get(left.text.toLowerCase())?.has(right.text.toLowerCase())) {
      continue
    }
    for (let i = wordAt[k] + 1; i < wordAt[k + 1]; i++) {
      tokens[i].marked = true
    }
  }
}

/** Marks every literal occurrence of the query inside the still unmarked stretches. */
function markLiteralQuery(segments: HighlightSegment[], query: string) {
  const needle = query.toLowerCase()
  if (!needle) return segments

  return segments.flatMap<HighlightSegment>((segment) => {
    if (segment.marked) return segment

    const parts: HighlightSegment[] = []
    const haystack = segment.text.toLowerCase()
    let pos = 0

    for (
      let at = haystack.indexOf(needle);
      at !== -1;
      at = haystack.indexOf(needle, pos)
    ) {
      if (at > pos) {
        parts.push({ text: segment.text.slice(pos, at), marked: false })
      }
      parts.push({
        text: segment.text.slice(at, at + needle.length),
        marked: true,
      })
      pos = at + needle.length
    }

    if (parts.length === 0) return segment
    if (pos < segment.text.length) {
      parts.push({ text: segment.text.slice(pos), marked: false })
    }
    return parts
  })
}

function mergeAdjacent(tokens: Token[]): HighlightSegment[] {
  const segments: HighlightSegment[] = []

  for (const token of tokens) {
    const last = segments[segments.length - 1]
    if (last && last.marked === token.marked) {
      last.text += token.text
      continue
    }
    segments.push({ text: token.text, marked: token.marked })
  }

  return segments
}

/**
 * Splits text into runs that should and should not be highlighted.
 *
 * Set `escaped` for a server side highlight fragment, whose text the search index HTML
 * escaped. Complete values arrive raw, and decoding those would turn an entity that the
 * game text itself contains into the character it stands for.
 */
export function resolveHighlightSegments(
  text: string,
  query: string,
  escaped = false,
): HighlightSegment[] {
  if (!text) return []

  const tokens = parseMarks(text, escaped ? unescapeHtmlEntities : (s) => s)
  if (query) {
    expandMarks(tokens, query)
  }

  return markLiteralQuery(mergeAdjacent(tokens), query)
}
