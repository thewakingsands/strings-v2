import { Button, ButtonGroup } from '@blueprintjs/core'
import styled from '@emotion/styled'

const PagerContainer = styled.div({
  textAlign: 'center',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  gap: '12px',
})

const PageInfo = styled.span({
  fontSize: '14px',
  minWidth: '80px',
  textAlign: 'center',
})

export interface IPagerProps {
  current: number
  hasMore: boolean
  onPageChange: (page: number) => void
}

export function Pager(props: IPagerProps) {
  const isFirstPage = props.current <= 1
  const hasNoMore = !props.hasMore

  return (
    <PagerContainer>
      <ButtonGroup>
        <Button
          icon="double-chevron-left"
          text="首页"
          onClick={() => props.onPageChange(1)}
          disabled={isFirstPage}
        />
        <Button
          icon="chevron-left"
          text="上一页"
          onClick={() => props.onPageChange(props.current - 1)}
          disabled={isFirstPage}
        />
      </ButtonGroup>
      <PageInfo>第 {props.current} 页</PageInfo>
      <ButtonGroup>
        <Button
          endIcon="chevron-right"
          text="下一页"
          onClick={() => props.onPageChange(props.current + 1)}
          disabled={hasNoMore}
        />
      </ButtonGroup>
    </PagerContainer>
  )
}
