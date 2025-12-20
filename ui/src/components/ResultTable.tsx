import { Classes, Colors, HTMLTable } from '@blueprintjs/core'
import styled from '@emotion/styled'
import type { StringItem } from '@/search/interface'
import { highlightText } from '@/utils/highlight'
import { defaultDisplayLanguages, languageMap } from '@/utils/language'
import { useScrollIntoView } from '../utils/useScrollIntoView'

const HighlightedTbody = styled.tbody({
  em: {
    textDecoration: 'none',
    backgroundColor: Colors.GOLD5,
    fontStyle: 'normal',
  },
  td: {
    whiteSpace: 'pre-wrap',
    wordBreak: 'break-word',
    cursor: 'auto',
  },
  'td:nth-of-type(1)': {
    fontSize: 12,
    fontFamily: 'monospace',
  },
  '.highlight-row td': {
    backgroundColor: `#fef4a8 !important`,
  },
})

const StyledHtmlTable = styled(HTMLTable)({
  minWidth: 700,
  width: '100%',
  tableLayout: 'fixed',
  [`&.${Classes.HTML_TABLE}.${Classes.HTML_TABLE_STRIPED} tbody tr:hover td`]: {
    backgroundColor: Colors.LIGHT_GRAY4,
  },
  borderCollapse: 'collapse',
  'th, tr, td': {
    border: '1px solid #fff',
  },
})

const ScrollableContainer = styled.div({
  width: '100%',
  overflowX: 'auto',
})

const LinkButton = styled.button({
  padding: 0,
  margin: 0,
  border: 0,
  backgroundColor: 'transparent',
  color: Colors.BLUE3,
  cursor: 'pointer',
  '&:hover': {
    textDecoration: 'underline',
  },
})

export interface IResultTableProps {
  items: StringItem[]
  onContextButtonClick?: (item: StringItem) => void
  keyword: string
  highlightItem?: Pick<StringItem, 'sheet' | 'rowId'>
}

export function ResultTable(props: IResultTableProps) {
  const { items, keyword } = props
  useScrollIntoView('.highlight-row', [props.highlightItem])

  const displayLanguages = defaultDisplayLanguages.map((lang) => ({
    label: languageMap[lang],
    value: lang,
  }))

  return (
    <ScrollableContainer>
      <StyledHtmlTable striped>
        <colgroup>
          <col style={{ width: '120px' }} />
          <col
            span={displayLanguages.length}
            style={{
              width: `calc((100% - 120px) / ${displayLanguages.length})`,
            }}
          />
        </colgroup>
        <thead>
          <tr>
            <th>位置</th>
            {displayLanguages.map((lang) => (
              <th key={lang.value}>{lang.label}</th>
            ))}
          </tr>
        </thead>
        <HighlightedTbody className={Classes.TEXT_LARGE}>
          {items.map((item, idx) => (
            <tr
              key={`${item.sheet}-${item.rowId}-${idx}`}
              className={
                item.sheet === props.highlightItem?.sheet &&
                item.rowId === props.highlightItem?.rowId
                  ? 'highlight-row'
                  : ''
              }
            >
              <td>
                {item.sheet}#{item.rowId}
                <br />
                <LinkButton onClick={() => props.onContextButtonClick?.(item)}>
                  搜索上下文
                </LinkButton>
              </td>
              {displayLanguages.map((lang) => {
                const value = item.values[lang.value]
                return (
                  <td key={lang.value}>
                    {value ? highlightText(value, keyword) : ''}
                  </td>
                )
              })}
            </tr>
          ))}
        </HighlightedTbody>
      </StyledHtmlTable>
    </ScrollableContainer>
  )
}
