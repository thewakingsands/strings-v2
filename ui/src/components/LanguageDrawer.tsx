import { Button, Colors, Drawer, Position } from '@blueprintjs/core'
import styled from '@emotion/styled'
import type { PropsWithChildren } from 'react'

const Sheet = styled.div({
  padding: '6px 12px 20px',
  // Keeps the last row clear of the home indicator on phones that have one.
  paddingBottom: 'max(20px, env(safe-area-inset-bottom))',
})

const Grabber = styled.div({
  width: 36,
  height: 4,
  borderRadius: 2,
  backgroundColor: Colors.LIGHT_GRAY1,
  margin: '4px auto 6px',
})

const Header = styled.div({
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  padding: '0 0 2px 4px',
  h4: {
    margin: 0,
    fontSize: 16,
    fontWeight: 600,
  },
})

export interface ILanguageDrawerProps {
  title: string
  isOpen: boolean
  onClose: () => void
}

/** The phone's counterpart of a popover: a sheet from the bottom edge, in reach of a thumb. */
export function LanguageDrawer({
  title,
  isOpen,
  onClose,
  children,
}: PropsWithChildren<ILanguageDrawerProps>) {
  return (
    <Drawer
      isOpen={isOpen}
      onClose={onClose}
      position={Position.BOTTOM}
      size="auto"
      style={{ borderRadius: '14px 14px 0 0' }}
    >
      <Sheet>
        <Grabber />
        <Header>
          <h4>{title}</h4>
          <Button variant="minimal" intent="primary" onClick={onClose}>
            完成
          </Button>
        </Header>
        {children}
      </Sheet>
    </Drawer>
  )
}
