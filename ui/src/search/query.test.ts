import { describe, expect, it } from 'vitest'
import { splitSheetTokens } from './query'

describe('splitSheetTokens', () => {
  it.each([
    ['onion', 'onion', ''],
    ['onion filename:Item', 'onion', 'Item'],
    [
      'filename:quest/* 覚えていろ filename:cut_scene/*',
      '覚えていろ',
      'quest/*,cut_scene/*',
    ],
    ['覚えていろ -filename:Item,Action', '覚えていろ', '-Item,-Action'],
    [
      '覚えていろ filename:quest/* -filename:Item,Action',
      '覚えていろ',
      'quest/*,-Item,-Action',
    ],
    ['a filename: b', 'a b', ''],
    ['a  filename:Item   b', 'a b', 'Item'],
    ['filename:Item,,Action', '', 'Item,Action'],
    ['-onion', '-onion', ''],
  ])('%s', (input, q, sheet) => {
    expect(splitSheetTokens(input)).toEqual({ q, sheet })
  })
})
