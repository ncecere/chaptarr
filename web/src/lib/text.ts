/**
 * Turns provider HTML (book descriptions, author bios) into plain text.
 * The markup is parsed into an inert document (no scripts run, no resources
 * load) and only its text content is kept, so the result is safe to render as
 * an ordinary React text node. Paragraph and line breaks become newlines.
 */
export function htmlToText(html: string | null | undefined): string {
  if (!html) return ''
  const withBreaks = html
    .replace(/<br\s*\/?>/gi, '\n')
    .replace(/<\/(p|div|li|h[1-6]|blockquote)>/gi, '\n\n')
    .replace(/<li[^>]*>/gi, '• ')
  const doc = new DOMParser().parseFromString(withBreaks, 'text/html')
  const text = doc.body.textContent ?? ''
  return text
    .replace(/\r/g, '')
    .replace(/[ \t]+\n/g, '\n')
    .replace(/\n{3,}/g, '\n\n')
    .trim()
}

/** Copies text to the clipboard, also on plain-http origins where the async API is unavailable. */
export async function copyText(text: string): Promise<boolean> {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch {
    // fall through to the legacy path
  }
  const ta = document.createElement('textarea')
  ta.value = text
  ta.setAttribute('readonly', '')
  ta.style.position = 'fixed'
  ta.style.opacity = '0'
  document.body.appendChild(ta)
  ta.select()
  let ok = false
  try {
    ok = document.execCommand('copy')
  } catch {
    ok = false
  }
  document.body.removeChild(ta)
  return ok
}
