import { useEffect } from 'react'

export function useScrollIntoView(selector: string, deps: unknown[]) {
  useEffect(() => {
    if (selector) {
      requestAnimationFrame(() => {
        const dom = document.querySelector(selector)
        if (dom) {
          document.scrollingElement?.scrollTo({
            top: (dom as HTMLElement).offsetTop - 100,
            behavior: 'smooth',
          })
        }
      })
    }
  }, [selector, ...deps])
}
