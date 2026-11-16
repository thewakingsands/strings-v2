import { Button, InputGroup, Tooltip } from '@blueprintjs/core'
import styled from '@emotion/styled'
import type { CSSProperties } from 'react'

export interface ISearchFieldProps {
  keyword: string
  onKeywordChange: (keyword: string) => void
  /** Reads the keyword as a bleve query string instead of plain words. */
  advanced: boolean
  onAdvancedChange: (advanced: boolean) => void
  className?: string
  style?: CSSProperties
}

const SyntaxList = styled.dl({
  margin: 0,
  display: 'grid',
  gridTemplateColumns: 'auto auto',
  columnGap: 12,
  rowGap: 2,
  dt: { fontFamily: 'monospace' },
  dd: { margin: 0 },
})

const syntaxExamples: [string, string][] = [
  ['*鲈*', '通配，能搜到单个汉字'],
  ['"bass fish"', '短语'],
  ['+a +b', '必须同时包含'],
  ['-a', '排除'],
  ['chs:鲈鱼', '只搜某个语言'],
  ['sheet:Item', '只搜某个表'],
  ['/暗.+/', '正则，匹配单个词'],
  ['fish~1', '模糊匹配'],
]

// Stripped out of the query by splitSheetTokens before it is sent, so these work in
// plain search too, unlike sheet: above.
const sheetTokenExamples: [string, string][] = [
  ['filename:quest/*', '只搜匹配的表（* ? 通配）'],
  ['filename:Item,Action', '只搜其中任一个表'],
  ['-filename:Item', '排除某个表'],
]

function syntaxRows(examples: [string, string][]) {
  return examples.map(([syntax, meaning]) => (
    <div key={syntax} style={{ display: 'contents' }}>
      <dt>{syntax}</dt>
      <dd>{meaning}</dd>
    </div>
  ))
}

const advancedHelp = (
  <div>
    <div>高级查询（bleve query string）</div>
    <SyntaxList>{syntaxRows(syntaxExamples)}</SyntaxList>
    <div style={{ marginTop: 8 }}>两种模式都可用</div>
    <SyntaxList>{syntaxRows(sheetTokenExamples)}</SyntaxList>
  </div>
)

export function SearchField(props: ISearchFieldProps) {
  return (
    <InputGroup
      className={props.className}
      size="large"
      value={props.keyword}
      onChange={(e) => props.onKeywordChange(e.target.value)}
      leftIcon="search"
      placeholder={props.advanced ? '高级查询，如 chs:*鲈* -en:fish' : '搜索'}
      autoFocus
      rightElement={
        <>
          {props.keyword && (
            <Button
              icon="small-cross"
              variant="minimal"
              onClick={() => props.onKeywordChange('')}
            />
          )}
          <Tooltip content={advancedHelp} placement="bottom-end">
            <Button
              icon="code"
              variant="minimal"
              active={props.advanced}
              intent={props.advanced ? 'primary' : 'none'}
              aria-pressed={props.advanced}
              aria-label="高级查询"
              onClick={() => props.onAdvancedChange(!props.advanced)}
            />
          </Tooltip>
        </>
      }
    />
  )
}
