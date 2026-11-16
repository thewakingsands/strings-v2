import { Button, Colors, Popover } from '@blueprintjs/core'
import styled from '@emotion/styled'
import { languageMap, languageOptions } from '@/utils/language'
import { usePointerReorder } from '@/utils/usePointerReorder'
import { MOBILE_BREAKPOINT } from './breakpoints'
import { LanguageList } from './LanguageList'

const ADD_SLOT_WIDTH = 34
const HIDE_BUTTON_SPACE = 24

const HeaderRow = styled.div({
  display: 'flex',
  flexDirection: 'row',
  borderBottom: '2px solid #e1e8ed',
  backgroundColor: '#f7f9fa',
  fontWeight: 600,
  fontSize: 12,
  color: '#5c7080',
  // Sticks under the search field, whose height App measures into the variable. It must stay
  // opaque: the rows scroll underneath it.
  position: 'sticky',
  top: 'var(--results-header-top, 50px)',
  zIndex: 15,
  [`@media (max-width: ${MOBILE_BREAKPOINT - 1}px)`]: {
    display: 'none',
  },
})

const HeaderCellPosition = styled.div({
  order: -1,
  flex: '0 0 120px',
  minWidth: 0,
  padding: '10px 12px',
  borderRight: '1px solid #e1e8ed',
})

const HeaderCellLang = styled.div<{ $last: boolean; $dragging: boolean }>(
  ({ $last, $dragging }) => ({
    flex: 1,
    minWidth: 0,
    padding: '10px 12px',
    // Not :last-of-type: the ＋ is a sibling of these cells, so once it is there the last language
    // header is no longer the last element and the rule would silently stop matching — leaving the
    // header one 1px border wider than every body row.
    borderRight: $last ? 'none' : '1px solid #e1e8ed',
    position: 'relative',
    cursor: 'grab',
    userSelect: 'none',
    ...($dragging && {
      cursor: 'grabbing',
      backgroundColor: '#e7effa',
      color: Colors.BLUE2,
    }),
  }),
)

/**
 * The inset for the ✕ and the ＋ goes here rather than on the cell: the cells are `flex: 1`, whose
 * basis is 0, so with border-box padding is added on top of the space they were given and padding
 * one of them would make its column a different width from the body cell underneath it.
 */
const HeaderLabel = styled.span<{ $inset: number }>(({ $inset }) => ({
  display: 'block',
  minWidth: 0,
  overflow: 'hidden',
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
  paddingRight: $inset,
}))

// Always rendered rather than shown on hover: a control nobody can find is the problem this card
// set out to fix. Faint enough to stay out of the way until it is looked for.
const HideButton = styled.button<{ $right: number }>(({ $right }) => ({
  position: 'absolute',
  right: $right,
  top: '50%',
  transform: 'translateY(-50%)',
  width: 18,
  height: 18,
  padding: 0,
  margin: 0,
  border: 0,
  borderRadius: '50%',
  display: 'grid',
  placeItems: 'center',
  backgroundColor: 'transparent',
  color: '#c2ccd6',
  cursor: 'pointer',
  fontSize: 12,
  lineHeight: 1,
  '&:hover': {
    backgroundColor: '#e1e6ea',
    color: Colors.GRAY1,
  },
}))

const AddSlot = styled.div({
  position: 'absolute',
  right: 0,
  top: 0,
  bottom: 0,
  width: ADD_SLOT_WIDTH,
  display: 'grid',
  placeItems: 'center',
  borderLeft: '1px solid #e1e8ed',
  backgroundColor: '#f7f9fa',
  // The cells carry an `order`, which also paints them in that order, after this slot.
  zIndex: 1,
})

export interface IResultTableHeaderProps {
  displayLanguages: string[]
  onHideLanguage: (code: string) => void
  onToggleLanguage: (code: string) => void
  onReorderDisplayLanguages: (from: number, to: number) => void
}

export function ResultTableHeader({
  displayLanguages,
  onHideLanguage,
  onToggleLanguage,
  onReorderDisplayLanguages,
}: IResultTableHeaderProps) {
  const { draggingIndex, handleProps } = usePointerReorder({
    axis: 'x',
    count: displayLanguages.length,
    onMove: onReorderDisplayLanguages,
  })

  const canRemove = displayLanguages.length > 1
  const hasHiddenLanguages = displayLanguages.length < languageOptions.length
  const lastIndex = displayLanguages.length - 1

  // Rendered in one fixed DOM order and placed by `order`, as LanguageList does: moving the
  // node under the pointer mid-drag would drop its pointer capture.
  const shown = languageOptions
    .map((option) => option.value)
    .filter((lang) => displayLanguages.includes(lang))

  return (
    <HeaderRow>
      <HeaderCellPosition>位置</HeaderCellPosition>
      {shown.map((lang) => {
        const index = displayLanguages.indexOf(lang)
        const props = handleProps(index)
        const isLast = index === lastIndex
        const clearsAddSlot = isLast && hasHiddenLanguages
        const label = languageMap[lang as keyof typeof languageMap] ?? lang
        return (
          <HeaderCellLang
            key={lang}
            style={{ order: index }}
            $last={isLast}
            $dragging={draggingIndex === index}
            {...props}
            // The ✕ sits inside the cell; pressing it hides the column and must not start a drag.
            onPointerDown={(event) => {
              if ((event.target as Element).closest('button')) return
              props.onPointerDown(event)
            }}
            // Arrow keys move the column only from the cell itself, not from the ✕ inside it.
            onKeyDown={(event) => {
              if (event.target !== event.currentTarget) return
              props.onKeyDown(event)
            }}
            tabIndex={0}
          >
            <HeaderLabel
              $inset={
                (canRemove ? HIDE_BUTTON_SPACE : 0) +
                (clearsAddSlot ? ADD_SLOT_WIDTH : 0)
              }
            >
              {label}
            </HeaderLabel>
            {canRemove && (
              <HideButton
                type="button"
                $right={clearsAddSlot ? ADD_SLOT_WIDTH + 6 : 6}
                title={`隐藏${label}列`}
                aria-label={`隐藏${label}列`}
                onClick={() => onHideLanguage(lang)}
              >
                ✕
              </HideButton>
            )}
          </HeaderCellLang>
        )
      })}
      {hasHiddenLanguages && (
        <AddSlot>
          <Popover
            placement="bottom-end"
            content={
              <LanguageList
                selected={displayLanguages}
                onToggle={onToggleLanguage}
                onReorder={onReorderDisplayLanguages}
                caption="勾上的语言显示为结果的列，上下顺序就是列的左右顺序。"
              />
            }
          >
            <Button
              icon="plus"
              variant="minimal"
              size="small"
              aria-label="添加语言列"
            />
          </Popover>
        </AddSlot>
      )}
    </HeaderRow>
  )
}
