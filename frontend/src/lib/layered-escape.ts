// reka-ui 2.10.4 keeps a closing layer mounted until its CSS animation ends
// (Presence waits for a changed animation-name). While it is mounted it is still
// the top dismissable layer, so the next Escape would be swallowed. Finish that
// layer and hand the same keypress to whatever is underneath.
export function installLayeredEscape() {
  let forwarding = false

  window.addEventListener(
    'keydown',
    (event) => {
      if (forwarding || event.key !== 'Escape' || event.repeat) {
        return
      }
      const layers = document.querySelectorAll<HTMLElement>('[data-dismissable-layer]')
      const top = layers[layers.length - 1]
      if (!top || top.dataset.state !== 'closed') {
        return
      }
      const animationName = getComputedStyle(top).animationName
      top.style.animationName = 'none'
      top.dispatchEvent(new AnimationEvent('animationend', { animationName }))
      event.stopImmediatePropagation()
      window.setTimeout(() => {
        forwarding = true
        window.dispatchEvent(
          new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }),
        )
        forwarding = false
      }, 0)
    },
    true,
  )
}
