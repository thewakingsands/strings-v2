import { Classes, Colors } from '@blueprintjs/core'
import styled from '@emotion/styled'
import { HttpError } from '@/search/query'
import { StyledNonIdealState } from './StyledNonIdealState'

const ErrorNonIdealState = styled(StyledNonIdealState)({
  [`.${Classes.ICON}`]: {
    color: Colors.RED3,
  },
})

export function SearchError(props: { error: Error }) {
  const { error } = props
  return (
    <ErrorNonIdealState
      icon="error"
      title={
        error instanceof HttpError && error.status === 400
          ? '查询有误'
          : '搜索出错'
      }
      description={error.message}
    />
  )
}
