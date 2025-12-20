import { Classes } from '@blueprintjs/core'
import styled from '@emotion/styled'
import type { PropsWithChildren } from 'react'
import { FixedWidthContainer } from './FixedWidthContainer'

const PaddedDiv = styled.div({
  paddingTop: 50,
})

const PaddedContainer = styled(FixedWidthContainer)({
  paddingTop: 12,
})

export function MainContainer({ children }: PropsWithChildren) {
  return (
    <PaddedDiv className={Classes.RUNNING_TEXT}>
      <PaddedContainer>{children}</PaddedContainer>
    </PaddedDiv>
  )
}
