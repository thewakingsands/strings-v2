/** Where an item dragged `offset` px along the axis lands, given a uniform `step` per slot. */
export function reorderTarget(
  from: number,
  offset: number,
  step: number,
  count: number,
): number {
  if (step <= 0 || count < 1) return from
  const to = from + Math.round(offset / step)
  return Math.min(Math.max(to, 0), count - 1)
}
