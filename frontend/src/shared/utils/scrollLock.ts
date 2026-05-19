type PrevOverflow = {
  htmlOverflow: string
  bodyOverflow: string
  htmlPaddingRight: string
  bodyPaddingRight: string
}

let lockCount = 0
let prev: PrevOverflow | null = null

export function lockPageScroll(): void {
  lockCount += 1
  if (lockCount !== 1) return

  const html = document.documentElement
  const body = document.body
  const scrollbarW = window.innerWidth - html.clientWidth

  prev = {
    htmlOverflow: html.style.overflow,
    bodyOverflow: body.style.overflow,
    htmlPaddingRight: html.style.paddingRight,
    bodyPaddingRight: body.style.paddingRight
  }

  html.style.overflow = 'hidden'
  body.style.overflow = 'hidden'
  if (scrollbarW > 0) {
    html.style.paddingRight = `${scrollbarW}px`
    body.style.paddingRight = `${scrollbarW}px`
  }
}

export function unlockPageScroll(): void {
  if (lockCount <= 0) return
  lockCount -= 1
  if (lockCount !== 0) return
  if (!prev) return

  const html = document.documentElement
  const body = document.body
  html.style.overflow = prev.htmlOverflow
  body.style.overflow = prev.bodyOverflow
  html.style.paddingRight = prev.htmlPaddingRight
  body.style.paddingRight = prev.bodyPaddingRight
  prev = null
}

