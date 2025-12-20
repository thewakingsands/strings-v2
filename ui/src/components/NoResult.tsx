import { StyledNonIdealState } from './StyledNonIdealState'

export function NoResult(props: { keyword: string }) {
  return (
    <StyledNonIdealState
      icon="search"
      description={
        props.keyword ? `没有找到“${props.keyword}”` : '请输入关键字'
      }
    />
  )
}
