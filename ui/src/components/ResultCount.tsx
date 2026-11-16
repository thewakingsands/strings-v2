import { Colors } from '@blueprintjs/core'
import styled from '@emotion/styled'
import { forwardRef, type PropsWithChildren } from 'react'
import { MOBILE_BREAKPOINT } from './breakpoints'
import { staleStyle } from './StaleResults'

const text = {
  fontSize: 12,
  color: Colors.GRAY1,
  fontVariantNumeric: 'tabular-nums',
} as const

// Whether the count fits beside a centred pager depends on both widths, which change with the
// total and the page. So the count takes no width in the row, and the pager carries an invisible
// copy of it on either side: the row breaks exactly when those three no longer fit, and
// wrap-reverse then puts the count's line under the pager's. Baseline alignment, unlike
// centring, ignores the margins that give the stacked lines their spacing.
const Row = styled.div({
  display: 'flex',
  flexWrap: 'wrap-reverse',
  alignItems: 'baseline',
})

const Count = styled.div<{ $busy: boolean }>(({ $busy }) => ({
  ...text,
  flex: 'none',
  width: 0,
  marginBottom: 6,
  // Painted over the pager's area, which spans the row beneath it, so the count stays selectable.
  position: 'relative',
  ...($busy && staleStyle),
}))

const PagerArea = styled.div({
  flex: 'auto',
  minWidth: 0,
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  marginBottom: 12,
})

const Ghost = styled.div({
  ...text,
  flex: '0 1 auto',
  minWidth: 0,
  overflow: 'clip',
  visibility: 'hidden',
})

// Inset to start under the first label of whichever header is below it. The right side is the
// least room the count leaves before the pager.
const Label = styled.span({
  whiteSpace: 'nowrap',
  padding: '0 16px 0 12px',
  [`@media (max-width: ${MOBILE_BREAKPOINT - 1}px)`]: {
    paddingLeft: 2,
  },
})

/** An advanced query's count, beside the pager when there is room for both and under it if not. */
export const ResultCount = forwardRef<
  HTMLDivElement,
  PropsWithChildren<{ total: number; busy: boolean }>
>(function ResultCount({ total, busy, children }, ref) {
  const label = <Label>共 {total.toLocaleString()} 条</Label>
  return (
    <Row ref={ref}>
      <Count $busy={busy}>{label}</Count>
      <PagerArea>
        <Ghost aria-hidden>{label}</Ghost>
        {children}
        <Ghost aria-hidden>{label}</Ghost>
      </PagerArea>
    </Row>
  )
})
