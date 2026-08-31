import { Colors } from '@blueprintjs/core'
import styled from '@emotion/styled'
import { useState } from 'react'
import { type LanguageOption, orderActiveFirst } from '@/utils/language'

const Row = styled.div({
  display: 'flex',
  gap: 6,
  alignItems: 'center',
  flexWrap: 'wrap',
})

const RowLabel = styled.span({
  color: Colors.GRAY1,
  fontSize: 12,
  flex: '0 0 auto',
})

const Chip = styled.button<{ $selected: boolean; $dropBefore: boolean }>(
  ({ $selected, $dropBefore }) => ({
    display: 'inline-flex',
    alignItems: 'center',
    gap: 5,
    border: `1px solid ${$selected ? Colors.BLUE3 : '#e1e8ed'}`,
    borderRadius: 14,
    padding: '3px 11px',
    fontSize: 13,
    lineHeight: '20px',
    fontFamily: 'inherit',
    cursor: 'pointer',
    userSelect: 'none',
    backgroundColor: $selected ? Colors.BLUE3 : Colors.WHITE,
    color: $selected ? Colors.WHITE : Colors.GRAY1,
    '&:hover': {
      borderColor: $selected ? Colors.BLUE2 : '#b6c2cf',
    },
    ...($dropBefore && {
      boxShadow: `-3px 0 0 ${Colors.BLUE3}`,
    }),
  }),
)

const Rank = styled.span({
  fontSize: 11,
  minWidth: 15,
  height: 15,
  borderRadius: 8,
  backgroundColor: 'rgba(255, 255, 255, 0.28)',
  display: 'grid',
  placeItems: 'center',
})

export interface ILanguageChipsProps {
  options: LanguageOption[]
  /** Ordered; for the query languages that order is the priority the server boosts by. */
  selected: string[]
  onToggle: (code: string) => void
  onReorder?: (from: number, to: number) => void
  showRank?: boolean
  label?: string
}

export function LanguageChips({
  options,
  selected,
  onToggle,
  onReorder,
  showRank,
  label,
}: ILanguageChipsProps) {
  const [dragFrom, setDragFrom] = useState<number | null>(null)
  const [dropAt, setDropAt] = useState<number | null>(null)

  const codes = orderActiveFirst(
    selected,
    options.map((option) => option.value),
  )
  const labels = new Map(options.map((option) => [option.value, option.label]))

  const endDrag = () => {
    setDragFrom(null)
    setDropAt(null)
  }

  return (
    <Row>
      {label && <RowLabel>{label}</RowLabel>}
      {codes.map((code) => {
        const rank = selected.indexOf(code)
        const isSelected = rank !== -1
        // Only the picked chips carry a position, so only they can be dragged or dropped on.
        const draggable = isSelected && onReorder !== undefined
        return (
          <Chip
            key={code}
            type="button"
            $selected={isSelected}
            $dropBefore={dropAt === rank && dragFrom !== null}
            aria-pressed={isSelected}
            draggable={draggable}
            onClick={() => onToggle(code)}
            onDragStart={
              draggable
                ? (event) => {
                    setDragFrom(rank)
                    event.dataTransfer.effectAllowed = 'move'
                    event.dataTransfer.setData('text/plain', code)
                  }
                : undefined
            }
            onDragEnd={draggable ? endDrag : undefined}
            onDragOver={
              draggable
                ? (event) => {
                    if (dragFrom === null) return
                    event.preventDefault()
                    setDropAt(rank)
                  }
                : undefined
            }
            onDrop={
              draggable
                ? (event) => {
                    event.preventDefault()
                    if (dragFrom !== null) onReorder?.(dragFrom, rank)
                    endDrag()
                  }
                : undefined
            }
          >
            {labels.get(code) ?? code}
            {showRank && isSelected && <Rank>{rank + 1}</Rank>}
          </Chip>
        )
      })}
    </Row>
  )
}
