import { describe, expect, it } from 'vitest'
import {
  isFragmentTruncated,
  markedWords,
  resolveExpandedSegments,
  resolveHighlightSegments,
} from './highlightSegments'

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

describe('markedWords', () => {
  it('collects the words the server marked', () => {
    const fragment =
      '…<mark>including</mark> <mark>without</mark> <mark>limitation</mark> the rights…'
    expect(markedWords(fragment)).toEqual([
      'including',
      'without',
      'limitation',
    ])
  })

  it('lowercases and de-duplicates', () => {
    expect(markedWords('<mark>Onion</mark> and <mark>onion</mark>')).toEqual([
      'onion',
    ])
  })

  it('decodes entities before splitting when escaped', () => {
    // The server marked the word inside curly quotes it had escaped.
    expect(markedWords('the &#34;<mark>Software</mark>&#34;', true)).toEqual([
      'software',
    ])
  })

  it('splits a multi word mark', () => {
    expect(markedWords('<mark>wizard eggplant</mark>')).toEqual([
      'wizard',
      'eggplant',
    ])
  })

  it('returns nothing for a fragment with no marks', () => {
    expect(markedWords('Thavnairian onion, no marks here')).toEqual([])
    expect(markedWords('')).toEqual([])
  })
})

describe('resolveExpandedSegments', () => {
  const seeds = ['including', 'without', 'limitation', 'rights']

  it('highlights the seed words in the complete value', () => {
    const value = 'text including without limitation the rights to use'
    expect(
      resolveExpandedSegments(
        value,
        'including without limitation the rights',
        seeds,
      )
        .filter((s) => s.marked)
        .map((s) => s.text),
    ).toEqual(['including without limitation the rights'])
  })

  it('does not mark a seed that is only part of a longer word', () => {
    // `the` must not light up inside `there`.
    const marked = resolveExpandedSegments('there and then', 'the cat', ['the'])
      .filter((s) => s.marked)
      .map((s) => s.text)
    expect(marked).toEqual([])
  })

  it('marks every occurrence, not just the first', () => {
    const marked = resolveExpandedSegments(
      'onion soup and onion pie',
      'onion',
      ['onion'],
    )
      .filter((s) => s.marked)
      .map((s) => s.text)
    expect(marked).toEqual(['onion', 'onion'])
  })

  it('pulls in a stop word the query puts between two seeds', () => {
    // `to` is a stop word so the server never marked it, but the query has it between
    // afford and lower.
    const marked = resolveExpandedSegments(
      'we can afford to lower our guard',
      'afford to lower',
      ['afford', 'lower'],
    )
      .filter((s) => s.marked)
      .map((s) => s.text)
    expect(marked).toEqual(['afford to lower'])
  })

  it('falls back to the literal query when there are no seeds', () => {
    const marked = resolveExpandedSegments('a onion b', 'onion', [])
      .filter((s) => s.marked)
      .map((s) => s.text)
    expect(marked).toEqual(['onion'])
  })

  it('rebuilds the value exactly', () => {
    const value = 'text including without limitation the rights to use'
    expect(
      resolveExpandedSegments(value, 'including the rights', seeds)
        .map((s) => s.text)
        .join(''),
    ).toBe(value)
  })

  it('handles an empty value', () => {
    expect(resolveExpandedSegments('', 'onion', ['onion'])).toEqual([])
  })
})

describe('isFragmentTruncated', () => {
  it('is false when the fragment is the whole value', () => {
    expect(
      isFragmentTruncated('To be continued...', 'To be continued...'),
    ).toBe(false)
  })

  it('is false once the mark tags are discounted', () => {
    expect(isFragmentTruncated('<mark>onion</mark> soup', 'onion soup')).toBe(
      false,
    )
  })

  it('is false once the entities are decoded', () => {
    // Left escaped, this fragment measures longer than the raw value it came from, which
    // would read as "not truncated" for the wrong reason.
    expect(
      isFragmentTruncated(
        'the <mark>wives</mark>&#39; tale',
        "the wives' tale",
      ),
    ).toBe(false)
  })

  it('is true when the fragment covers only part of the value', () => {
    const value = 'a'.repeat(500)
    expect(isFragmentTruncated(`…${'a'.repeat(200)}…`, value)).toBe(true)
  })

  it('is not fooled by an ellipsis inside the text itself', () => {
    // Chinese punctuation, not the fragmenter's separator.
    expect(isFragmentTruncated('……大人？！', '……大人？！')).toBe(false)
  })

  it('is false for an empty fragment or value', () => {
    expect(isFragmentTruncated('', 'onion')).toBe(false)
    expect(isFragmentTruncated('onion', '')).toBe(false)
  })
})
