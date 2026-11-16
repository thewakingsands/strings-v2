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
  // Before anything is typed the focused search box already asks for input.
  if (!props.keyword) {
    return null
  }
  return (
    <StyledNonIdealState
      icon="search"
      description={`没有找到“${props.keyword}”`}
    />
  )
}
