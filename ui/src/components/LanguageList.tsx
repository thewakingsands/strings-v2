import { Checkbox, Classes, Colors, Icon } from '@blueprintjs/core'
import styled from '@emotion/styled'
import {
  inlineNames,
  languageOptions,
  orderActiveFirst,
} from '@/utils/language'
import { usePointerReorder } from '@/utils/usePointerReorder'

const LIFT_SHADOW =
  '0 1px 2px rgba(17, 20, 24, 0.1), 0 6px 18px rgba(17, 20, 24, 0.14)'

const Container = styled.div<{ $touch: boolean }>(({ $touch }) => ({
  padding: $touch ? 0 : 6,
  minWidth: $touch ? 0 : 240,
}))

const Caption = styled.p<{ $touch: boolean }>(({ $touch }) => ({
  margin: 0,
  padding: $touch ? '10px 4px 0' : '8px 8px 6px',
  maxWidth: $touch ? 'none' : 260,
  fontSize: 12,
  lineHeight: 1.5,
  color: Colors.GRAY1,
  // A caption a character or two too long for one line would leave them alone on the second;
  // evening out the lines keeps that from depending on how long the text happens to be. Chinese
  // would otherwise break between any two characters, splitting a word, so it breaks at
  // punctuation, and anywhere only when a clause cannot fit a line by itself.
  textWrap: 'balance',
  wordBreak: 'keep-all',
  overflowWrap: 'anywhere',
}))

const List = styled.div({
  display: 'flex',
  flexDirection: 'column',
})

const Row = styled.div<{ $touch: boolean; $lifted: boolean }>(
  ({ $touch, $lifted }) => ({
    display: 'flex',
    alignItems: 'center',
    gap: 4,
    height: $touch ? 48 : 34,
    padding: '0 6px 0 2px',
    borderRadius: 4,
    fontSize: $touch ? 15 : 14,
    backgroundColor: Colors.WHITE,
    position: 'relative',
    '&:hover': {
      backgroundColor: Colors.LIGHT_GRAY5,
    },
    ...($lifted && {
      boxShadow: LIFT_SHADOW,
      zIndex: 1,
      '&:hover': { backgroundColor: Colors.WHITE },
    }),
  }),
)

const Handle = styled.span<{ $touch: boolean }>(({ $touch }) => ({
  width: $touch ? 34 : 22,
  height: $touch ? 44 : 28,
  flex: 'none',
  display: 'grid',
  placeItems: 'center',
  borderRadius: 3,
  color: Colors.GRAY4,
  cursor: 'grab',
  // Lets a finger on the handle drag the row instead of scrolling the drawer; anywhere else on
  // the row still scrolls.
  touchAction: 'none',
  '&:hover': {
    color: Colors.GRAY1,
  },
  '&:active': {
    cursor: 'grabbing',
  },
}))

const HandleSpace = styled.span<{ $touch: boolean }>(({ $touch }) => ({
  width: $touch ? 34 : 22,
  flex: 'none',
}))

const Option = styled(Checkbox)<{ $unselected: boolean }>(
  ({ $unselected }) => ({
    '&&': {
      flex: 1,
      minWidth: 0,
      alignSelf: 'stretch',
      display: 'flex',
      alignItems: 'center',
      margin: 0,
      color: $unselected ? Colors.GRAY1 : 'inherit',
    },
    // Blueprint lifts the box 3px to sit on an inline text baseline; centred in a flex row, the
    // lift leaves it above the name and the drag handle instead.
    [`&& .${Classes.CONTROL_INDICATOR}`]: {
      marginTop: 0,
    },
  }),
)

const Name = styled.span({
  flex: 1,
  minWidth: 0,
  display: 'flex',
  alignItems: 'center',
  gap: 8,
})

const Divider = styled.div({
  height: 1,
  margin: '4px 6px',
  backgroundColor: Colors.LIGHT_GRAY2,
})

export interface ILanguageListProps {
  /** Ordered; the order is the one the handles drag. */
  selected: string[]
  onToggle: (code: string) => void
  onReorder: (from: number, to: number) => void
  /** Shown below the rows only when more than one is picked; the list does not know which group it is. */
  caption?: string
  /** Touch sized rows, for the bottom drawers. */
  touch?: boolean
}

const labels = new Map(
  languageOptions.map((option) => [option.value, option.label]),
)

/** The picker behind both groups: what is picked, in which order, and what is not. */
export function LanguageList({
  selected,
  onToggle,
  onReorder,
  caption,
  touch = false,
}: ILanguageListProps) {
  const { draggingIndex, handleProps } = usePointerReorder({
    axis: 'y',
    count: selected.length,
    onMove: onReorder,
  })

  const ordered = orderActiveFirst(
    selected,
    languageOptions.map((option) => option.value),
  )
  const canUntick = selected.length > 1

  return (
    <Container $touch={touch}>
      {/* The rows stay in one DOM order and are laid out by `order`: React moves a reordered
          keyed node, and moving it drops the pointer capture and focus of the handle being
          dragged mid-gesture. */}
      <List>
        {languageOptions.map(({ value: code }) => {
          const position = ordered.indexOf(code)
          const rank = selected.indexOf(code)
          const isSelected = rank !== -1
          const label = labels.get(code) ?? code
          return (
            <Row
              key={code}
              style={{ order: isSelected ? position : position + 1 }}
              $touch={touch}
              $lifted={isSelected && draggingIndex === rank}
            >
              {isSelected ? (
                <Handle
                  $touch={touch}
                  role="button"
                  tabIndex={0}
                  aria-label={`调整${inlineNames(label)}的顺序`}
                  {...handleProps(rank)}
                >
                  <Icon icon="drag-handle-vertical" size={14} />
                </Handle>
              ) : (
                <HandleSpace $touch={touch} />
              )}
              <Option
                $unselected={!isSelected}
                checked={isSelected}
                // Emptying the group has no use, so the last one picked cannot be unticked.
                disabled={isSelected && !canUntick}
                onChange={() => onToggle(code)}
                labelElement={<Name>{label}</Name>}
              />
            </Row>
          )
        })}
        {selected.length < languageOptions.length && (
          <Divider style={{ order: selected.length }} />
        )}
      </List>
      {/* Below every row, so its coming and going never shifts one. The bottom drawer grows
          upwards, though, so there it keeps its space while hidden or the rows would rise. */}
      {caption &&
        (selected.length > 1 ? (
          <Caption $touch={touch}>{caption}</Caption>
        ) : (
          touch && (
            <Caption $touch aria-hidden style={{ visibility: 'hidden' }}>
              {caption}
            </Caption>
          )
        ))}
    </Container>
  )
}
