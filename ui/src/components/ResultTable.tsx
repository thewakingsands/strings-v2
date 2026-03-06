import { Colors } from '@blueprintjs/core'
import styled from '@emotion/styled'
import type { StringItem } from '@/search/interface'
import { highlightText } from '@/utils/highlight'
import { languageMap } from '@/utils/language'
import { useScrollIntoView } from '../utils/useScrollIntoView'

const MOBILE_BREAKPOINT = 768

const ListContainer = styled.div({
  width: '100%',
  minWidth: 0,
  display: 'flex',
  flexDirection: 'column',
})

const HeaderRow = styled.div({
  display: 'flex',
  flexDirection: 'row',
  borderBottom: '2px solid #e1e8ed',
  backgroundColor: '#f7f9fa',
  fontWeight: 600,
  fontSize: 12,
  color: '#5c7080',
  [`@media (max-width: ${MOBILE_BREAKPOINT - 1}px)`]: {
    display: 'none',
  },
})

const HeaderCellPosition = styled.div({
  flex: '0 0 120px',
  minWidth: 0,
  padding: '10px 12px',
  borderRight: '1px solid #e1e8ed',
})

const HeaderCellLang = styled.div({
  flex: 1,
  minWidth: 0,
  padding: '10px 12px',
  borderRight: '1px solid #e1e8ed',
  '&:last-of-type': {
    borderRight: 'none',
  },
})

const ItemRow = styled.div<{ $highlight?: boolean }>(({ $highlight }) => ({
  display: 'flex',
  flexDirection: 'row',
  minWidth: 0,
  borderBottom: '1px solid #e1e8ed',
  '&:hover': {
    backgroundColor: Colors.LIGHT_GRAY4,
  },
  ...($highlight && {
    backgroundColor: '#fef4a8',
  }),
  [`@media (max-width: ${MOBILE_BREAKPOINT - 1}px)`]: {
    flexDirection: 'column',
    border: '1px solid #e1e8ed',
    borderRadius: 8,
    marginBottom: 12,
    padding: 12,
    boxShadow: '0 1px 3px rgba(0,0,0,0.08)',
    '&:hover': {
      backgroundColor: 'transparent',
    },
    ...($highlight && {
      backgroundColor: '#fef4a8',
      borderColor: '#d99e0b',
    }),
  },
}))

const CellPosition = styled.div({
  flex: '0 0 120px',
  minWidth: 0,
  padding: '10px 12px',
  borderRight: '1px solid #e1e8ed',
  fontSize: 12,
  fontFamily: 'monospace',
  whiteSpace: 'pre-wrap',
  wordBreak: 'break-word',
  [`@media (max-width: ${MOBILE_BREAKPOINT - 1}px)`]: {
    flex: 'none',
    width: '100%',
    borderRight: 'none',
    borderBottom: '1px solid #e1e8ed',
    marginBottom: 8,
    paddingTop: 0,
    paddingBottom: 8,
    paddingLeft: 0,
    paddingRight: 0,
  },
})

const CellLang = styled.div({
  flex: 1,
  minWidth: 0,
  padding: '10px 12px',
  borderRight: '1px solid #e1e8ed',
  whiteSpace: 'pre-wrap',
  wordBreak: 'break-word',
  fontSize: 14,
  '&:last-of-type': {
    borderRight: 'none',
  },
  '& em': {
    textDecoration: 'none',
    backgroundColor: Colors.GOLD5,
    fontStyle: 'normal',
  },
  [`@media (max-width: ${MOBILE_BREAKPOINT - 1}px)`]: {
    display: 'flex',
    flexDirection: 'row',
    flex: 'none',
    width: '100%',
    borderRight: 'none',
    paddingLeft: 0,
    paddingRight: 0,
    paddingTop: 6,
    paddingBottom: 0,
    '&::before': {
      content: 'attr(data-label)',
      fontWeight: 600,
      display: 'inline-block',
      minWidth: 72,
      marginRight: 8,
      color: '#5c7080',
    },
  },
})

const ScrollableContainer = styled.div({
  width: '100%',
  minWidth: 0,
  overflowX: 'auto',
  [`@media (max-width: ${MOBILE_BREAKPOINT - 1}px)`]: {
    overflowX: 'visible',
  },
})

const LinkButton = styled.button({
  padding: 0,
  margin: 0,
  border: 0,
  backgroundColor: 'transparent',
  color: Colors.BLUE3,
  cursor: 'pointer',
  fontSize: 12,
  '&:hover': {
    textDecoration: 'underline',
  },
})

export interface IResultTableProps {
  items: StringItem[]
  onContextButtonClick?: (item: StringItem) => void
  keyword: string
  highlightItem?: Pick<StringItem, 'sheet' | 'rowId'>
  displayLanguages: string[]
}

export function ResultTable(props: IResultTableProps) {
  const { items, keyword, displayLanguages } = props
  useScrollIntoView('[data-highlight-row="true"]', [props.highlightItem])

  return (
    <ScrollableContainer>
      <ListContainer>
        <HeaderRow>
          <HeaderCellPosition>位置</HeaderCellPosition>
          {displayLanguages.map((lang) => (
            <HeaderCellLang key={lang}>
              {languageMap[lang as keyof typeof languageMap]}
            </HeaderCellLang>
          ))}
        </HeaderRow>
        {items.map((item, idx) => {
          const isHighlight =
            item.sheet === props.highlightItem?.sheet &&
            item.rowId === props.highlightItem?.rowId
          return (
            <ItemRow
              key={`${item.sheet}-${item.rowId}-${idx}`}
              $highlight={isHighlight}
              data-highlight-row={isHighlight ? 'true' : undefined}
            >
              <CellPosition>
                {item.sheet}#{item.rowId}
                <br />
                <LinkButton onClick={() => props.onContextButtonClick?.(item)}>
                  搜索上下文
                </LinkButton>
              </CellPosition>
              {displayLanguages.map((lang) => {
                const value = item.values[lang]
                const label = languageMap[lang as keyof typeof languageMap]
                return (
                  <CellLang key={lang} data-label={label}>
                    <div>{value ? highlightText(value, keyword) : ''}</div>
                  </CellLang>
                )
              })}
            </ItemRow>
          )
        })}
      </ListContainer>
    </ScrollableContainer>
  )
}
