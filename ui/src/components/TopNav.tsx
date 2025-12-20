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

const WIKI_USER_URL =
  'https://ff14.huijiwiki.com/wiki/%E7%94%A8%E6%88%B7:%E4%BA%91%E6%B3%BD%E5%AE%9B%E9%A3%8E'
const WEIBO_USER_URL = 'https://weibo.com/u/6364253854'

const MAP_URL = 'https://map.wakingsands.com'

function MoreToolsMenu() {
  return (
    <Menu>
      <MenuDivider title="其它工具" />
      <MenuItem
        text="交互地图"
        icon="map"
        onClick={() => window.open(MAP_URL, '_blank')}
      />
      <MenuDivider title="关于" />
      <MenuItem
        text="微博 @云泽宛风"
        icon="person"
        onClick={() => window.open(WEIBO_USER_URL, '_blank')}
      />
      <MenuItem
        text="维基 用户:云泽宛风"
        icon="edit"
        onClick={() => window.open(WIKI_USER_URL, '_blank')}
      />
      <MenuItem
        text="闲聊群 612370226"
        onClick={() => navigator.clipboard.writeText('612370226')}
        icon="chat"
      />
    </Menu>
  )
}
