import { Spinner } from '@blueprintjs/core'
import { keyframes } from '@emotion/react'
import styled from '@emotion/styled'
import type { PropsWithChildren } from 'react'

// Most answers arrive within this, and dimming for them would only make the page blink.
const SHOW_DELAY = '150ms'

const fadeIn = keyframes({
  from: { opacity: 0 },
  to: { opacity: 1 },
})

const Area = styled.div({
  position: 'relative',
})

/** Also for whatever outside the rows describes them, so that it fades along with them. */
export const staleStyle = {
  opacity: 0.4,
  transition: `opacity 150ms ${SHOW_DELAY}`,
  pointerEvents: 'none',
} as const

const Rows = styled.div<{ $busy: boolean }>(({ $busy }) =>
  $busy ? staleStyle : {},
)

const SpinnerLayer = styled.div({
  position: 'absolute',
  inset: 0,
  animation: `${fadeIn} 150ms ${SHOW_DELAY} both`,
})

// Sticky inside a layer as tall as the rows, so it stays in view at whatever point of a long
// page the reader turned it from.
const CenteredSpinner = styled(Spinner)({
  position: 'sticky',
  top: 'calc(50vh - 25px)',
})

/**
 * Keeps the rows already on screen while the next ones load, so the page keeps its height: the
 * scroll position and the scrollbar stay where they are instead of collapsing to a spinner.
 */
export function StaleResults({
  busy,
  children,
}: PropsWithChildren<{ busy: boolean }>) {
  return (
    <Area aria-busy={busy}>
      <Rows $busy={busy}>{children}</Rows>
      {busy && (
        <SpinnerLayer>
          <CenteredSpinner size={50} />
        </SpinnerLayer>
      )}
    </Area>
  )
}
