import { Menu, MenuItem } from '@blueprintjs/core'
import { languageOptions, orderActiveFirst } from '@/utils/language'

export interface IDisplayLanguageMenuProps {
  /** Ordered: this is the column order, and the menu lists the shown ones in it. */
  displayLanguages: string[]
  onToggle: (code: string) => void
}

/** The content behind both the navbar language button and the ＋ at the end of the header row. */
export function DisplayLanguageMenu({
  displayLanguages,
  onToggle,
}: IDisplayLanguageMenuProps) {
  const codes = orderActiveFirst(
    displayLanguages,
    languageOptions.map((option) => option.value),
  )
  const labels = new Map(
    languageOptions.map((option) => [option.value, option.label]),
  )

  return (
    <Menu>
      {codes.map((code) => {
        const shown = displayLanguages.includes(code)
        return (
          <MenuItem
            key={code}
            text={labels.get(code) ?? code}
            // The tick comes from `selected` alone; passing an icon as well draws a second one.
            roleStructure="listoption"
            selected={shown}
            // Emptying the table has no use, so the last column left cannot be unticked.
            disabled={shown && displayLanguages.length < 2}
            onClick={() => onToggle(code)}
            // Picking several languages in one go should not close the menu after the first.
            shouldDismissPopover={false}
          />
        )
      })}
    </Menu>
  )
}
