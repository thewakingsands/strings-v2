import { Button, Colors } from '@blueprintjs/core'
import styled from '@emotion/styled'
import { useState } from 'react'
import { languageMap } from '@/utils/language'
import { LanguageDrawer } from './LanguageDrawer'
import { LanguageList } from './LanguageList'

const Bar = styled.div({
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  padding: '0 2px 10px',
  fontSize: 12,
  color: Colors.GRAY1,
})

const Names = styled.span({
  flex: 1,
  minWidth: 0,
  overflow: 'hidden',
  whiteSpace: 'nowrap',
  textOverflow: 'ellipsis',
  color: Colors.DARK_GRAY5,
})

export interface IDisplayLanguageBarProps {
  displayLanguages: string[]
  onToggle: (code: string) => void
  onReorder: (from: number, to: number) => void
}

/** The phone's stand-in for the table header, which a stack of cards does not have. */
export function DisplayLanguageBar({
  displayLanguages,
  onToggle,
  onReorder,
}: IDisplayLanguageBarProps) {
  const [open, setOpen] = useState(false)

  return (
    <Bar>
      <span>显示</span>
      <Names>
        {displayLanguages
          .map((lang) => languageMap[lang as keyof typeof languageMap] ?? lang)
          .join(' · ')}
      </Names>
      <Button
        variant="minimal"
        intent="primary"
        size="small"
        onClick={() => setOpen(true)}
      >
        调整
      </Button>
      <LanguageDrawer
        title="显示语言"
        isOpen={open}
        onClose={() => setOpen(false)}
      >
        <LanguageList
          touch
          selected={displayLanguages}
          onToggle={onToggle}
          onReorder={onReorder}
        />
      </LanguageDrawer>
    </Bar>
  )
}
