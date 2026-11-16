import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import {
  defaultDisplayLanguages,
  defaultQueryLanguages,
  inlineNames,
  orderActiveFirst,
  readStoredDisplayLanguages,
  readStoredQueryLanguages,
  writeStoredDisplayLanguages,
  writeStoredQueryLanguages,
} from './language'

/** Enough of the Storage interface for the two keys this module touches. */
function fakeStorage(entries: Record<string, string> = {}) {
  const store = new Map(Object.entries(entries))
  return {
    getItem: (key: string) => store.get(key) ?? null,
    setItem: (key: string, value: string) => {
      store.set(key, value)
    },
    removeItem: (key: string) => {
      store.delete(key)
    },
    clear: () => {
      store.clear()
    },
    key: (index: number) => [...store.keys()][index] ?? null,
    get length() {
      return store.size
    },
  } as Storage
}

function install(storage: Storage) {
  Object.defineProperty(globalThis, 'localStorage', {
    value: storage,
    configurable: true,
    writable: true,
  })
}

const queryKey = 'xivstrings.queryLanguages'
const displayKey = 'xivstrings.displayLanguages'

beforeEach(() => {
  install(fakeStorage())
})

afterEach(() => {
  Reflect.deleteProperty(globalThis, 'localStorage')
})

describe('language preferences round trip', () => {
  it('gives the defaults when nothing was ever stored', () => {
    expect(readStoredQueryLanguages()).toEqual([...defaultQueryLanguages])
    expect(readStoredDisplayLanguages()).toEqual([...defaultDisplayLanguages])
  })

  it('reads back the selection that was written', () => {
    writeStoredQueryLanguages(['ja', 'en'])
    writeStoredDisplayLanguages(['en', 'ja'])

    expect(readStoredQueryLanguages()).toEqual(['ja', 'en'])
    expect(readStoredDisplayLanguages()).toEqual(['en', 'ja'])
  })

  it('keeps the query order, which is the priority the server boosts by', () => {
    writeStoredQueryLanguages(['ja', 'chs'])
    expect(readStoredQueryLanguages()).toEqual(['ja', 'chs'])

    writeStoredQueryLanguages(['chs', 'ja'])
    expect(readStoredQueryLanguages()).toEqual(['chs', 'ja'])
  })

  it('keeps the display order, which is the column order the user dragged', () => {
    // STR-23 moves the display languages into the result table header and lets them be
    // reordered, so this must come back exactly as stored rather than re-sorted.
    install(fakeStorage({ [displayKey]: JSON.stringify(['ja', 'chs', 'en']) }))
    expect(readStoredDisplayLanguages()).toEqual(['ja', 'chs', 'en'])

    writeStoredDisplayLanguages(['en', 'ja', 'chs'])
    expect(readStoredDisplayLanguages()).toEqual(['en', 'ja', 'chs'])
  })

  it('validates an out of order value without sorting the survivors', () => {
    install(
      fakeStorage({ [displayKey]: JSON.stringify(['ja', 'klingon', 'chs']) }),
    )
    expect(readStoredDisplayLanguages()).toEqual(['ja', 'chs'])
  })

  it('drops the tc column for good once it is removed', () => {
    // The complaint behind this card: 繁中 had to be closed on every visit.
    writeStoredDisplayLanguages(['chs', 'en', 'ja'])
    expect(readStoredDisplayLanguages()).not.toContain('tc')
  })
})

describe('stored values that cannot be trusted', () => {
  it('ignores a language the app no longer has', () => {
    install(
      fakeStorage({
        [queryKey]: JSON.stringify(['chs', 'klingon']),
        [displayKey]: JSON.stringify(['en', 'klingon']),
      }),
    )
    expect(readStoredQueryLanguages()).toEqual(['chs'])
    expect(readStoredDisplayLanguages()).toEqual(['en'])
  })

  it('ignores duplicates and non strings', () => {
    install(
      fakeStorage({ [queryKey]: JSON.stringify(['ja', 'ja', 7, null, 'en']) }),
    )
    expect(readStoredQueryLanguages()).toEqual(['ja', 'en'])
  })

  it('falls back to the defaults for a value that is not a list', () => {
    install(
      fakeStorage({
        [queryKey]: JSON.stringify({ chs: true }),
        [displayKey]: '"en"',
      }),
    )
    expect(readStoredQueryLanguages()).toEqual([...defaultQueryLanguages])
    expect(readStoredDisplayLanguages()).toEqual([...defaultDisplayLanguages])
  })

  it('falls back to the defaults for a value that does not parse', () => {
    install(fakeStorage({ [queryKey]: '{oh no', [displayKey]: '' }))
    expect(readStoredQueryLanguages()).toEqual([...defaultQueryLanguages])
    expect(readStoredDisplayLanguages()).toEqual([...defaultDisplayLanguages])
  })

  it('refuses an empty query language list, which the server answers 400 for', () => {
    install(fakeStorage({ [queryKey]: JSON.stringify([]) }))
    expect(readStoredQueryLanguages()).toEqual([...defaultQueryLanguages])
  })

  it('restores an empty display language list, which is a real choice', () => {
    // The display picker, unlike the query one, lets every language go.
    install(fakeStorage({ [displayKey]: JSON.stringify([]) }))
    expect(readStoredDisplayLanguages()).toEqual([])
  })
})

describe('orderActiveFirst', () => {
  const all = ['chs', 'tc', 'en', 'ja', 'ko']

  it('puts the active codes first, keeping the order they came in', () => {
    expect(orderActiveFirst(['ja', 'chs'], all)).toEqual([
      'ja',
      'chs',
      'tc',
      'en',
      'ko',
    ])
  })

  it('leaves the fixed order alone when nothing is active', () => {
    expect(orderActiveFirst([], all)).toEqual(all)
  })

  it('returns the active order unchanged when everything is active', () => {
    const active = ['ko', 'en', 'ja', 'tc', 'chs']
    expect(orderActiveFirst(active, all)).toEqual(active)
  })

  it('drops an active code the app does not have', () => {
    // The active list can come from storage, which may name a language since removed.
    expect(orderActiveFirst(['klingon', 'en'], all)).toEqual([
      'en',
      'chs',
      'tc',
      'ja',
      'ko',
    ])
  })
})

describe('storage that is not available', () => {
  const denied = {
    getItem() {
      throw new DOMException('denied', 'SecurityError')
    },
    setItem() {
      throw new DOMException('denied', 'SecurityError')
    },
  } as unknown as Storage

  it('still starts on the defaults when reading throws', () => {
    install(denied)
    expect(readStoredQueryLanguages()).toEqual([...defaultQueryLanguages])
    expect(readStoredDisplayLanguages()).toEqual([...defaultDisplayLanguages])
  })

  it('does not throw out of a write', () => {
    install(denied)
    expect(() => writeStoredQueryLanguages(['ja'])).not.toThrow()
    expect(() => writeStoredDisplayLanguages(['ja'])).not.toThrow()
  })

  it('does not throw when there is no storage at all', () => {
    Reflect.deleteProperty(globalThis, 'localStorage')
    expect(readStoredQueryLanguages()).toEqual([...defaultQueryLanguages])
    expect(() => writeStoredQueryLanguages(['ja'])).not.toThrow()
  })
})

describe('inlineNames', () => {
  it('leaves a name that is Chinese at both ends alone', () => {
    expect(inlineNames('简体中文')).toBe('简体中文')
  })

  it('pads a Latin name on both sides', () => {
    expect(inlineNames('English')).toBe(' English ')
  })

  it('pads only the end of a run that ends in Latin', () => {
    expect(inlineNames('简体中文、English')).toBe('简体中文、English ')
  })

  it('pads only the start of a run that starts in Latin', () => {
    expect(inlineNames('English、日本語')).toBe(' English、日本語')
  })
})
