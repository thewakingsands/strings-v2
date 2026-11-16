import { useEffect, useState } from 'react'
import { MOBILE_BREAKPOINT } from '@/components/breakpoints'

const query = `(max-width: ${MOBILE_BREAKPOINT - 1}px)`

/** The breakpoint the card layout's media queries use, for the choices CSS cannot make. */
export function useIsMobile(): boolean {
  const [isMobile, setIsMobile] = useState(
    () => window.matchMedia(query).matches,
  )

  useEffect(() => {
    const media = window.matchMedia(query)
    const update = () => setIsMobile(media.matches)
    update()
    media.addEventListener('change', update)
    return () => media.removeEventListener('change', update)
  }, [])

  return isMobile
}
