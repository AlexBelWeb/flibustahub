import MarkdownIt from 'markdown-it'

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
})

const allowed = /^(https?:|mailto:)/i

md.validateLink = (url) => allowed.test(url)

export function renderNote(source: string): string {
  return md.render(source)
}

export function isSafeExternalUrl(url: string): boolean {
  return allowed.test(url)
}
