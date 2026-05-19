export function normalizeMultilineText(input: string): string {
  let text = String(input ?? '')
  if (!text) return ''
  text = text.replace(/\r\n/g, '\n')
  text = text.replace(/\r/g, '\n')
  return text
}

export async function copyToClipboard(text: string): Promise<void> {
  const v = String(text ?? '')
  if (!v) return

  try {
    await navigator.clipboard.writeText(v)
    return
  } catch {
    const ta = document.createElement('textarea')
    ta.value = v
    ta.style.position = 'fixed'
    ta.style.left = '-9999px'
    ta.style.top = '0'
    document.body.appendChild(ta)
    ta.focus()
    ta.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(ta)
    if (!ok) throw new Error('copy_failed')
  }
}

export function sanitizeFileName(name: string): string {
  const raw = String(name ?? '')
  const safe = raw.replace(/[\\/:*?"<>|]/g, '-').replace(/\s+/g, ' ').trim()
  return safe || 'file'
}

export function downloadBlob(filename: string, blob: Blob) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.rel = 'noopener'
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

export function downloadTextFile(filename: string, text: string, mime = 'text/plain;charset=utf-8') {
  const v = String(text ?? '')
  if (!v) return
  downloadBlob(filename, new Blob([v], { type: mime }))
}
