import { Colors } from '@blueprintjs/core'
import styled from '@emotion/styled'
import { MOBILE_BREAKPOINT } from './breakpoints'
import { staleStyle } from './StaleResults'

// Inset to start under the first label of whichever header is below it.
const Line = styled.div<{ $busy: boolean }>(({ $busy }) => ({
  padding: '0 12px 6px',
  fontSize: 12,
  color: Colors.GRAY1,
  fontVariantNumeric: 'tabular-nums',
  [`@media (max-width: ${MOBILE_BREAKPOINT - 1}px)`]: {
    padding: '0 2px 6px',
  },
  ...($busy && staleStyle),
}))

export function ResultCount({ total, busy }: { total: number; busy: boolean }) {
  return <Line $busy={busy}>共 {total.toLocaleString()} 条</Line>
}
