import { type KeyboardEvent, type PointerEvent, useRef, useState } from 'react'
import { reorderTarget } from './reorderTarget'

interface DragState {
  pointerId: number
  index: number
  origin: number
  step: number
}

export interface PointerReorderHandleProps {
  onPointerDown: (event: PointerEvent<HTMLElement>) => void
  onPointerMove: (event: PointerEvent<HTMLElement>) => void
  onPointerUp: (event: PointerEvent<HTMLElement>) => void
  onPointerCancel: (event: PointerEvent<HTMLElement>) => void
  onKeyDown: (event: KeyboardEvent<HTMLElement>) => void
}

/**
 * Pointer events rather than HTML5 drag and drop, which touch screens never fire, so mouse, pen and
 * finger all take one path. The list is reordered while the pointer moves, which keeps the item
 * under it the one being dragged.
 */
export function usePointerReorder({
  axis,
  count,
  onMove,
}: {
  axis: 'x' | 'y'
  count: number
  onMove: (from: number, to: number) => void
}): {
  draggingIndex: number | null
  handleProps: (index: number) => PointerReorderHandleProps
} {
  const drag = useRef<DragState | null>(null)
  const [draggingIndex, setDraggingIndex] = useState<number | null>(null)

  const position = (event: PointerEvent<HTMLElement>) =>
    axis === 'x' ? event.clientX : event.clientY

  const end = (event: PointerEvent<HTMLElement>) => {
    if (drag.current?.pointerId !== event.pointerId) return
    drag.current = null
    setDraggingIndex(null)
  }

  const [backKey, forwardKey] =
    axis === 'x' ? ['ArrowLeft', 'ArrowRight'] : ['ArrowUp', 'ArrowDown']

  return {
    draggingIndex,
    handleProps: (index) => ({
      onPointerDown: (event) => {
        if (event.button !== 0) return
        // Without capture a finger that leaves the handle, as it must to move anything, stops
        // delivering events to it.
        event.currentTarget.setPointerCapture(event.pointerId)
        const rect = event.currentTarget.getBoundingClientRect()
        drag.current = {
          pointerId: event.pointerId,
          index,
          origin: position(event),
          step: axis === 'x' ? rect.width : rect.height,
        }
        setDraggingIndex(index)
      },
      onPointerMove: (event) => {
        const state = drag.current
        if (state?.pointerId !== event.pointerId) return
        const to = reorderTarget(
          state.index,
          position(event) - state.origin,
          state.step,
          count,
        )
        if (to === state.index) return
        onMove(state.index, to)
        // The item now sits in its new slot, so the offset is measured from there.
        state.origin += (to - state.index) * state.step
        state.index = to
        setDraggingIndex(to)
      },
      onPointerUp: end,
      onPointerCancel: end,
      onKeyDown: (event) => {
        const to =
          event.key === backKey
            ? index - 1
            : event.key === forwardKey
              ? index + 1
              : null
        if (to === null) return
        event.preventDefault()
        if (to >= 0 && to < count) onMove(index, to)
      },
    }),
  }
}
