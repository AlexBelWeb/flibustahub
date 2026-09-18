export interface HighlightChunk {
  text: string
  match: boolean
}

export function highlightChunks(text: string, query: string): HighlightChunk[] {
  const source = text || ''
  const tokens = query
    .trim()
    .split(/\s+/)
    .map((token) => fold(token.replace(/["^:*()%_]/g, '')))
    .filter((token) => token.length > 0)
  if (!source || tokens.length === 0) {
    return [{ text: source, match: false }]
  }
  const folded = fold(source)
  const marks = new Array<boolean>(source.length).fill(false)
  for (const token of tokens) {
    let from = 0
    while (from <= folded.length - token.length) {
      const at = folded.indexOf(token, from)
      if (at < 0) {
        break
      }
      for (let i = at; i < at + token.length && i < marks.length; i += 1) {
        marks[i] = true
      }
      from = at + token.length
    }
  }
  const chunks: HighlightChunk[] = []
  let current = ''
  let match = marks[0] ?? false
  for (let i = 0; i < source.length; i += 1) {
    const next = marks[i] ?? false
    if (next !== match && current.length > 0) {
      chunks.push({ text: current, match })
      current = ''
      match = next
    }
    current += source[i]
  }
  if (current.length > 0) {
    chunks.push({ text: current, match })
  }
  return chunks.length > 0 ? chunks : [{ text: source, match: false }]
}

function fold(value: string): string {
  return value.toLocaleLowerCase().replaceAll('ё', 'е').replaceAll('Ё', 'е')
}
