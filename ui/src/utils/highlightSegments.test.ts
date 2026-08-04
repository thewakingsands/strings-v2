import { describe, expect, it } from 'vitest'
import { resolveHighlightSegments } from './highlightSegments'

/** The highlighted runs, in order. */
function marked(text: string, query: string, escaped = false): string[] {
  return resolveHighlightSegments(text, query, escaped)
    .filter((segment) => segment.marked)
    .map((segment) => segment.text)
}

/** Everything concatenated back together, which must always equal the input text. */
function rebuilt(text: string, query: string, escaped = false): string {
  return resolveHighlightSegments(text, query, escaped)
    .map((segment) => segment.text)
    .join('')
}

describe('resolveHighlightSegments', () => {
  it('sews a phrase broken up by stop words back together', () => {
    // What the server returns: `we` and `to` are stop words, so they carry no term
    // location and never got marked.
    const text =
      '…d I always get up again. <mark>Anyway</mark>, we <mark>can</mark> ' +
      '<mark>hardly</mark> <mark>afford</mark> to <mark>lower</mark> our guard now.'
    const query = 'Anyway, we can hardly afford to lower'

    expect(marked(text, query)).toEqual([
      'Anyway, we can hardly afford to lower',
    ])
    expect(rebuilt(text, query)).toBe(
      '…d I always get up again. Anyway, we can hardly afford to lower our guard now.',
    )
  })

  it('takes in a run of stop words one after another', () => {
    expect(
      marked('<mark>out</mark> of the <mark>box</mark>', 'out of the box'),
    ).toEqual(['out of the box'])
  })

  it('expands from the right neighbour too', () => {
    expect(marked('to <mark>lower</mark>', 'afford to lower')).toEqual([
      'to lower',
    ])
  })

  it('does not bridge a gap the query does not have', () => {
    // `foo` is not what the query puts between `can` and `hardly`.
    expect(
      marked('<mark>can</mark> foo <mark>hardly</mark>', 'can hardly'),
    ).toEqual(['can', 'hardly'])
  })

  it('keeps hits apart when the query does not have them adjacent', () => {
    // A real second result for the same query: the line reads "we can afford", so `can`
    // and `afford` sit side by side here while the query has `hardly` between them.
    // Joining them would hide that the middle word is missing from this line.
    const text =
      '…it&#39;s not as if we <mark>can</mark> <mark>afford</mark> ' +
      'to <mark>lower</mark> our guard.'
    const query = 'Anyway, we can hardly afford to lower'

    expect(marked(text, query, true)).toEqual(['we can', 'afford to lower'])
  })

  it('stops at the ends of the query', () => {
    // `again` precedes the query's first word, `our` follows its last.
    expect(marked('again. <mark>Anyway</mark>', 'Anyway, we can')).toEqual([
      'Anyway',
    ])
    expect(marked('<mark>lower</mark> our guard', 'afford to lower')).toEqual([
      'lower',
    ])
  })

  it('leaves a single word query alone', () => {
    expect(marked('a <mark>wizard</mark> b', 'wizard')).toEqual(['wizard'])
  })

  it('handles a word that repeats in the query', () => {
    const text = '<mark>the</mark> cat and <mark>the</mark> dog'
    expect(marked(text, 'the cat and the dog')).toEqual(['the cat and the dog'])
  })

  it('reaches across punctuation between the words', () => {
    expect(marked('<mark>Anyway</mark>, we', 'Anyway, we')).toEqual([
      'Anyway, we',
    ])
  })

  it('decodes entities before splitting into words', () => {
    const text = 'a common <mark>wives</mark>&#39; tale'
    expect(marked(text, "wives' tale", true)).toEqual(["wives' tale"])
    expect(rebuilt(text, "wives' tale", true)).toBe("a common wives' tale")
  })

  it('leaves entities alone when the text is not escaped', () => {
    const text = '<mark>wives</mark>&#39; tale'
    expect(rebuilt(text, 'wives', false)).toBe('wives&#39; tale')
  })

  it('does not expand a cjk query, which is one word', () => {
    // The cjk analyzer has no stop words, so there is nothing to sew up, and 师傅大人
    // has no spaces to give the query any adjacency to go on.
    expect(marked('魔晶石<mark>师傅</mark>', '师傅大人')).toEqual(['师傅'])
  })

  it('marks every literal occurrence when the text has no marks', () => {
    // How a value from /api/items arrives: no server highlight at all.
    expect(marked('onion and onion soup', 'onion')).toEqual(['onion', 'onion'])
  })

  it('returns the text untouched when there is nothing to highlight', () => {
    expect(resolveHighlightSegments('', 'onion')).toEqual([])
    expect(resolveHighlightSegments('plain text', '')).toEqual([
      { text: 'plain text', marked: false },
    ])
    expect(marked('plain text', '!!!')).toEqual([])
    expect(rebuilt('plain text', '!!!')).toBe('plain text')
  })

  it('does not expand across a stemmed match, a known limitation', () => {
    // The server matched `hardliness` for the query `hardly` by stemming both, but the
    // display layer compares words literally, so it cannot tell they belong together
    // and leaves `we` alone. Sewing this up would mean stemming in the browser.
    expect(marked('<mark>hardliness</mark> we can', 'hardly we can')).toEqual([
      'hardliness',
    ])
  })
})
