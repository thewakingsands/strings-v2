import { describe, expect, it } from 'vitest'
import { reorderTarget } from './reorderTarget'

describe('reorderTarget', () => {
  it('stays put without movement', () => {
    expect(reorderTarget(2, 0, 34, 5)).toBe(2)
  })

  it('moves one slot forward', () => {
    expect(reorderTarget(1, 34, 34, 5)).toBe(2)
  })

  it('moves one slot back', () => {
    expect(reorderTarget(1, -34, 34, 5)).toBe(0)
  })

  it('stays put under half a slot', () => {
    expect(reorderTarget(1, 16, 34, 5)).toBe(1)
    expect(reorderTarget(1, -16, 34, 5)).toBe(1)
  })

  it('crosses over past half a slot', () => {
    expect(reorderTarget(1, 18, 34, 5)).toBe(2)
  })

  it('is clamped at both ends', () => {
    expect(reorderTarget(1, 1000, 34, 5)).toBe(4)
    expect(reorderTarget(1, -1000, 34, 5)).toBe(0)
  })

  it('stays put when a slot has no size', () => {
    expect(reorderTarget(3, 100, 0, 5)).toBe(3)
  })
})
