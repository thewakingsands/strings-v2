import { Colors } from '@blueprintjs/core'
import styled from '@emotion/styled'
import { StyledNonIdealState } from './StyledNonIdealState'

const ErrorNonIdealState = styled(StyledNonIdealState)({
  '.bp3-icon': {
    color: Colors.RED3,
  },
})

export function SearchError(props: { error: Error }) {
  return <ErrorNonIdealState icon="error" description={props.error.message} />
}
