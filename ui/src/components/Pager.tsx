import { Button, ButtonGroup } from '@blueprintjs/core'
import styled from '@emotion/styled'
import { useLayoutEffect, useRef, useState } from 'react'
import { flushSync } from 'react-dom'
import { pagerPages } from '../utils/pagerPages'

const PagerContainer = styled.div({
  textAlign: 'center',
})

// The nearest siblings are the ‹ and › buttons, which a narrow pager keeps.
const MAX_SIBLINGS = 3
const MIN_SIBLINGS = 1

export interface IPagerProps {
  current: number
  total: number
  onPageChange: (page: number) => void
}

export function Pager(props: IPagerProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const groupRef = useRef<HTMLDivElement>(null)
  const [room, setRoom] = useState(0)
  const [fit, setFit] = useState({ key: '', siblings: MAX_SIBLINGS })
  const key = `${props.current}/${props.total}/${room}`
  const siblings = fit.key === key ? fit.siblings : MAX_SIBLINGS

  // The parent's width is the room, not this container's: in ResultCount this container is only
  // as wide as its buttons, and the ghosts flanking it shrink away before the buttons overflow.
  // The parent's width never depends on the buttons, so refitting them cannot loop. Flushed so
  // that the refit lands before the resized frame is painted, not one frame after it.
  useLayoutEffect(() => {
    const parent = containerRef.current?.parentElement
    if (!parent) return
    const observer = new ResizeObserver(() =>
      flushSync(() => setRoom(parent.clientWidth)),
    )
    observer.observe(parent)
    return () => observer.disconnect()
  }, [])

  // How wide the buttons are depends on how many digits the page numbers have, so whether they
  // fit is known only after layout. Each page and room starts from the widest pager and gives up
  // the farthest siblings until it fits, all before paint.
  useLayoutEffect(() => {
    const group = groupRef.current
    const parent = containerRef.current?.parentElement
    if (!group || !parent || siblings <= MIN_SIBLINGS) return
    if (group.getBoundingClientRect().width > parent.clientWidth) {
      setFit({ key, siblings: siblings - 1 })
    }
  }, [key, siblings])

  const pages = pagerPages(props.current, props.total, siblings)

  const children = []

  let lastPage: number | undefined
  for (const page of pages) {
    if (lastPage !== undefined && page !== lastPage + 1) {
      children.push(<Button key={`before${page}`} text="…" disabled minimal />)
    }
    children.push(renderPageButton(page, props))

    lastPage = page
  }

  return (
    <PagerContainer ref={containerRef}>
      <ButtonGroup ref={groupRef}>{children}</ButtonGroup>
    </PagerContainer>
  )
}

function renderPageButton(page: number, props: IPagerProps) {
  const handleClick = () => props.onPageChange(page)
  if (page === props.current) {
    return (
      <Button key={page} text={page} disabled minimal onClick={handleClick} />
    )
  }
  if (page === props.current - 1) {
    return (
      <Button
        key={page}
        icon="chevron-left"
        aria-label="上一页"
        onClick={handleClick}
      />
    )
  }
  if (page === props.current + 1) {
    return (
      <Button
        key={page}
        endIcon="chevron-right"
        aria-label="下一页"
        onClick={handleClick}
      />
    )
  }
  if (page === 1) {
    return (
      <Button
        key={page}
        text="1"
        icon="double-chevron-left"
        onClick={handleClick}
      />
    )
  }
  if (page === props.total) {
    return (
      <Button
        key={page}
        text={page}
        endIcon="double-chevron-right"
        onClick={handleClick}
      />
    )
  }
  return <Button key={page} text={page} onClick={handleClick} />
}
