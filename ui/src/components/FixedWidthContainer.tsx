import styled from '@emotion/styled'
import type React from 'react'

const MarginDiv = styled.div({
  maxWidth: 2160,
  margin: '0 auto',
  padding: '0 10px',
})

export function FixedWidthContainer({
  children,
  className,
}: {
  children: React.ReactNode
  className?: string
}) {
  return <MarginDiv className={className}>{children}</MarginDiv>
}
