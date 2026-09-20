export function eventsOn(name: string, callback: (...data: unknown[]) => void): () => void {
  const off = window.runtime?.EventsOn(name, callback)
  return typeof off === 'function' ? off : () => {}
}

export function quitApp(): void {
  window.runtime?.Quit()
}

export function openExternalUrl(url: string): void {
  window.runtime?.BrowserOpenURL(url)
}
