import { Button, ButtonGroup } from '@blueprintjs/core'
import styled from '@emotion/styled'

const PagerContainer = styled.div({
  textAlign: 'center',
})

export interface IPagerProps {
  current: number
  total: number
  onPageChange: (page: number) => void
}

export function Pager(props: IPagerProps) {
  const pages = Array.from(
    new Set<number>([
      1,
      props.total,
      props.current - 1,
      props.current - 2,
      props.current - 3,
      props.current,
      props.current + 1,
      props.current + 2,
      props.current + 3,
    ]),
  )
    .filter((x) => x > 0 && x <= props.total)
    .sort((a, b) => a - b)

  const children = []

  let lastPage: number | undefined
  for (const page of pages) {
    if (lastPage !== undefined && page !== lastPage + 1) {
      children.push(
        <Button key={`before${page}`} text="..." disabled minimal />,
      )
    }
    children.push(renderPageButton(page, props))

    lastPage = page
  }

  return (
    <PagerContainer>
      <ButtonGroup>{children}</ButtonGroup>
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
    return <Button key={page} icon="chevron-left" onClick={handleClick} />
  }
  if (page === props.current + 1) {
    return <Button key={page} endIcon="chevron-right" onClick={handleClick} />
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
