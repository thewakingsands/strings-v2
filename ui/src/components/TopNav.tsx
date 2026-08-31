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
import { DisplayLanguageMenu } from './DisplayLanguageMenu'
import { FixedWidthContainer } from './FixedWidthContainer'

export interface ITopNavProps {
  displayLanguages: string[]
  onToggleDisplayLanguage: (code: string) => void
}

export function TopNav(props: ITopNavProps) {
  return (
    <Navbar className={Classes.DARK} fixedToTop>
      <FixedWidthContainer>
        <Navbar.Group align={Alignment.LEFT}>
          <Navbar.Heading>XIV 文本检索</Navbar.Heading>
        </Navbar.Group>
        <Navbar.Group align={Alignment.RIGHT}>
          <DisplayLanguageButton {...props} />
          <MoreToolsButton />
        </Navbar.Group>
      </FixedWidthContainer>
    </Navbar>
  )
}

/**
 * The result table's header carries the same control, but only where there is a table: the landing
 * page and an empty result set have none. The navbar is fixed to the top and always rendered, so
 * putting it here too means the display languages can never become unreachable.
 */
function DisplayLanguageButton({
  displayLanguages,
  onToggleDisplayLanguage,
}: ITopNavProps) {
  return (
    <Popover
      placement="bottom-end"
      content={
        <DisplayLanguageMenu
          displayLanguages={displayLanguages}
          onToggle={onToggleDisplayLanguage}
        />
      }
    >
      <Button
        icon="translate"
        text={String(displayLanguages.length)}
        aria-label={`显示语言 (${displayLanguages.length})`}
      />
    </Popover>
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
