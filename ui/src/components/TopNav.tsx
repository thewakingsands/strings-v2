import {
  Alignment,
  Button,
  Classes,
  Menu,
  MenuDivider,
  MenuItem,
  Navbar,
  Popover,
} from '@blueprintjs/core'
import { FixedWidthContainer } from './FixedWidthContainer'

export function TopNav() {
  return (
    <Navbar className={Classes.DARK} fixedToTop>
      <FixedWidthContainer>
        <Navbar.Group align={Alignment.LEFT}>
          <Navbar.Heading>XIV 文本检索</Navbar.Heading>
        </Navbar.Group>
        <Navbar.Group align={Alignment.RIGHT}>
          <MoreToolsButton />
        </Navbar.Group>
      </FixedWidthContainer>
    </Navbar>
  )
}

function MoreToolsButton() {
  return (
    <Popover content={<MoreToolsMenu />}>
      <Button icon="more" intent="primary" />
    </Popover>
  )
}

const FFCAFE_URL = 'https://www.ffcafe.cn'

function MoreToolsMenu() {
  return (
    <Menu>
      <MenuDivider title="关于" />
      <MenuItem
        text="FFCAFE"
        icon="globe"
        onClick={() => window.open(FFCAFE_URL, '_blank')}
      />
      <MenuItem
        text="闲聊群 612370226"
        onClick={() => navigator.clipboard.writeText('612370226')}
        icon="chat"
      />
    </Menu>
  )
}
