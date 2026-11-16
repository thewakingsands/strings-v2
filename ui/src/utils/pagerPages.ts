/** The pages a pager offers: the first, the last, and `siblings` either side of `current`. */
export function pagerPages(
  current: number,
  total: number,
  siblings: number,
): number[] {
  const pages = new Set([1, total])
  for (let page = current - siblings; page <= current + siblings; page++) {
    pages.add(page)
  }
  return [...pages]
    .filter((page) => page > 0 && page <= total)
    .sort((a, b) => a - b)
}
