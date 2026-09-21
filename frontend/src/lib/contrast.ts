// Relative luminance and WCAG contrast. Callers resolve CSS colors to sRGB first.

function channelLinear(channel: number): number {
  const s = channel / 255
  return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
}

export function relativeLuminance(r: number, g: number, b: number): number {
  return 0.2126 * channelLinear(r) + 0.7152 * channelLinear(g) + 0.0722 * channelLinear(b)
}

export function contrastRatio(a: number, b: number): number {
  const lighter = Math.max(a, b)
  const darker = Math.min(a, b)
  return (lighter + 0.05) / (darker + 0.05)
}

let scratch: CanvasRenderingContext2D | null = null

function srgbBytes(color: string): [number, number, number] | null {
  if (!scratch) {
    const canvas = document.createElement('canvas')
    canvas.width = 1
    canvas.height = 1
    scratch = canvas.getContext('2d', { willReadFrequently: true })
  }
  if (!scratch || !color) {
    return null
  }
  scratch.clearRect(0, 0, 1, 1)
  scratch.fillStyle = '#000000'
  scratch.fillStyle = color
  scratch.fillRect(0, 0, 1, 1)
  const px = scratch.getImageData(0, 0, 1, 1).data
  return [px[0], px[1], px[2]]
}

export function contrastOf(foreground: string, background: string): number | null {
  const fg = srgbBytes(foreground)
  const bg = srgbBytes(background)
  if (!fg || !bg) {
    return null
  }
  return contrastRatio(relativeLuminance(...fg), relativeLuminance(...bg))
}
