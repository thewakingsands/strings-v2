import {
  type ILanguageFallbackProps,
  LanguageFallback,
} from './LanguageFallback'
import { StyledNonIdealState } from './StyledNonIdealState'

export function NoResult(props: {
  keyword: string
  /** Given only for a plain search's empty first page, the one case other languages can help. */
  fallback?: ILanguageFallbackProps
}) {
  if (props.fallback) {
    return <LanguageFallback {...props.fallback} />
  }
  return (
    <StyledNonIdealState
      icon="search"
      description={
        props.keyword ? `没有找到“${props.keyword}”` : '请输入关键字'
      }
    />
  )
}
