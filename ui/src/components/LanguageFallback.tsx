import {
  Button,
  Colors,
  NonIdealState,
  Spinner,
  Tooltip,
} from '@blueprintjs/core'
import styled from '@emotion/styled'
import type { LanguageHits } from '@/search/languageProbe'
import { inlineNames, languageMap } from '@/utils/language'

const State = styled(NonIdealState)({
  minHeight: 200,
  height: 'auto',
  padding: '40px 0 24px',
})

const Box = styled.div({
  width: '100%',
  maxWidth: 420,
  textAlign: 'left',
  border: `1px solid ${Colors.LIGHT_GRAY2}`,
  borderRadius: 8,
  overflow: 'hidden',
})

const BoxHeader = styled.div({
  padding: '8px 12px',
  fontSize: 12,
  color: Colors.GRAY1,
  backgroundColor: Colors.LIGHT_GRAY5,
  borderBottom: `1px solid ${Colors.LIGHT_GRAY2}`,
})

const Hit = styled.div({
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  padding: '8px 8px 8px 12px',
  borderBottom: `1px solid ${Colors.LIGHT_GRAY2}`,
  '&:last-child': {
    borderBottom: 0,
  },
})

const HitName = styled.span({
  flex: 1,
  minWidth: 0,
  color: Colors.DARK_GRAY1,
})

const HitCount = styled.span({
  color: Colors.GRAY1,
  fontSize: 13,
  fontVariantNumeric: 'tabular-nums',
  whiteSpace: 'nowrap',
})

const Probing = styled.div({
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  color: Colors.GRAY1,
  fontSize: 13,
})

function languageName(code: string): string {
  return languageMap[code as keyof typeof languageMap] ?? code
}

export interface ILanguageFallbackProps {
  keyword: string
  /** The languages the empty search looked in. */
  searchedLanguages: string[]
  loading: boolean
  ranked: LanguageHits[]
  onReplace: (lang: string) => void
  onAdd: (lang: string) => void
}

/** An empty result that says where the keyword would have been found instead. */
export function LanguageFallback({
  keyword,
  searchedLanguages,
  loading,
  ranked,
  onReplace,
  onAdd,
}: ILanguageFallbackProps) {
  const searched = inlineNames(searchedLanguages.map(languageName).join('、'))
  const description =
    !loading && ranked.length === 0
      ? `所有语言里都没有找到“${keyword}”`
      : searchedLanguages.length > 1
        ? `在${searched}里都没有找到“${keyword}”`
        : `在${searched}里没有找到“${keyword}”`

  return (
    <State
      icon="search"
      description={description}
      action={
        loading ? (
          <Probing>
            <Spinner size={16} />
            正在查其他语言…
          </Probing>
        ) : ranked.length > 0 ? (
          <Box>
            <BoxHeader>其他语言里有：</BoxHeader>
            {ranked.map(({ lang, total }, i) => {
              const name = languageName(lang)
              const only = `仅搜索${inlineNames(name)}`.trim()
              const also = `同时搜索${inlineNames(name)}`.trim()
              return (
                <Hit key={lang}>
                  <HitName>{name}</HitName>
                  <HitCount>{total.toLocaleString()} 条</HitCount>
                  <Tooltip content={only} placement="top">
                    <Button
                      icon="search"
                      size="small"
                      variant="minimal"
                      intent={i === 0 ? 'primary' : 'none'}
                      aria-label={only}
                      onClick={() => onReplace(lang)}
                    />
                  </Tooltip>
                  <Tooltip content={also} placement="top">
                    <Button
                      icon="plus"
                      size="small"
                      variant="minimal"
                      aria-label={also}
                      onClick={() => onAdd(lang)}
                    />
                  </Tooltip>
                </Hit>
              )
            })}
          </Box>
        ) : undefined
      }
    />
  )
}
