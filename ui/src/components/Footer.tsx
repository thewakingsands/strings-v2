import { Classes } from '@blueprintjs/core'

export function Footer() {
  return (
    <footer style={{ paddingTop: 12 }}>
      <p className={`${Classes.TEXT_MUTED} ${Classes.TEXT_SMALL}`}>
        FINAL FANTASY is a registered trademark of Square Enix Holdings Co.,
        Ltd.
        <br />
        Copyrighted Materials are extracted from FINAL FANTASY XIV © 2010 - 2019
        SQUARE ENIX CO., LTD. All Rights Reserved.
      </p>
    </footer>
  )
}
