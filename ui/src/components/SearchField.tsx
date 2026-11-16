import {
  Button,
  Classes,
  Colors,
  Icon,
  InputGroup,
  Popover,
  Tooltip,
} from '@blueprintjs/core'
import styled from '@emotion/styled'
import { type CSSProperties, useState } from 'react'
import { languageMap } from '@/utils/language'
import { useIsMobile } from '@/utils/useIsMobile'
import { LanguageDrawer } from './LanguageDrawer'
import { LanguageList } from './LanguageList'

export interface ISearchFieldProps {
  keyword: string
  onKeywordChange: (keyword: string) => void
  /** Reads the keyword as a bleve query string instead of plain words. */
  advanced: boolean
  onAdvancedChange: (advanced: boolean) => void
  /** Ordered by decreasing priority. */
  queryLanguages: string[]
  onToggleQueryLanguage: (code: string) => void
  onReorderQueryLanguages: (from: number, to: number) => void
  className?: string
  style?: CSSProperties
}

const QUERY_CAPTION = '匹配程度相同时，靠上语言里的结果排在前面。'

// Inset by the 1px the input draws its border with, which the scope button would cover otherwise.
const LeftSlot = styled.span({
  display: 'flex',
  alignItems: 'center',
  height: 38,
  margin: '1px 0 0 1px',
  [`& > .${Classes.ICON}`]: {
    margin: '0 10px 0 12px',
  },
})

const ScopeTrigger = styled.button({
  display: 'inline-flex',
  alignItems: 'center',
  gap: 6,
  height: '100%',
  padding: '0 10px 0 12px',
  border: 0,
  borderRight: `1px solid ${Colors.LIGHT_GRAY2}`,
  borderRadius: '3px 0 0 3px',
  backgroundColor: Colors.LIGHT_GRAY5,
  color: Colors.DARK_GRAY1,
  font: 'inherit',
  fontSize: 14,
  whiteSpace: 'nowrap',
  cursor: 'pointer',
  '&:hover': {
    backgroundColor: Colors.LIGHT_GRAY4,
  },
  [`& > .${Classes.ICON}`]: {
    color: Colors.GRAY1,
  },
})

const MoreCount = styled.span({
  color: Colors.BLUE3,
  fontSize: 12,
  fontWeight: 600,
})

function languageName(code: string): string {
  return languageMap[code as keyof typeof languageMap] ?? code
}

/** Names what is being searched, so the field itself says where the keyword is looked for. */
function ScopeButton({
  queryLanguages,
  onToggleQueryLanguage,
  onReorderQueryLanguages,
}: Pick<
  ISearchFieldProps,
  'queryLanguages' | 'onToggleQueryLanguage' | 'onReorderQueryLanguages'
>) {
  const isMobile = useIsMobile()
  const [drawerOpen, setDrawerOpen] = useState(false)

  const list = (
    <LanguageList
      selected={queryLanguages}
      onToggle={onToggleQueryLanguage}
      onReorder={onReorderQueryLanguages}
      caption={QUERY_CAPTION}
      touch={isMobile}
    />
  )
  const trigger = (
    <ScopeTrigger
      type="button"
      aria-haspopup="dialog"
      aria-label={`搜索语言：${queryLanguages.map(languageName).join('、')}`}
      onClick={isMobile ? () => setDrawerOpen(true) : undefined}
    >
      {languageName(queryLanguages[0])}
      {queryLanguages.length > 1 && (
        <MoreCount>+{queryLanguages.length - 1}</MoreCount>
      )}
      <Icon icon="caret-down" size={12} />
    </ScopeTrigger>
  )

  if (isMobile) {
    return (
      <>
        {trigger}
        <LanguageDrawer
          title="搜索语言"
          isOpen={drawerOpen}
          onClose={() => setDrawerOpen(false)}
        >
          {list}
        </LanguageDrawer>
      </>
    )
  }
  return (
    // The span lets the trigger fill the field's height; a popover target is inline otherwise.
    <Popover
      placement="bottom-start"
      content={list}
      targetTagName="span"
      targetProps={{ style: { height: '100%', display: 'flex' } }}
    >
      {trigger}
    </Popover>
  )
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
  ['chs:鲈鱼', '只搜某种语言'],
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
    <div style={{ marginTop: 8 }}>普通搜索也能用</div>
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
      leftElement={
        <LeftSlot>
          <ScopeButton
            queryLanguages={props.queryLanguages}
            onToggleQueryLanguage={props.onToggleQueryLanguage}
            onReorderQueryLanguages={props.onReorderQueryLanguages}
          />
          <Icon icon="search" />
        </LeftSlot>
      }
      placeholder={props.advanced ? '高级查询，如 chs:*鲈* -en:fish' : '搜索'}
      autoFocus
      rightElement={
        <>
          {props.keyword && (
            <Button
              icon="small-cross"
              variant="minimal"
              aria-label="清空"
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
