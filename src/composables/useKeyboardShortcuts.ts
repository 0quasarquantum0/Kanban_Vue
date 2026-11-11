import { onKeyStroke } from '@vueuse/core'

type ShortcutHandler = (event: KeyboardEvent) => void

function isTypingTarget(target: EventTarget | null): boolean {
  return (
    target instanceof HTMLElement &&
    (target.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName))
  )
}

function isModalOpen(): boolean {
  return document.querySelector('[role="dialog"][aria-modal="true"]') !== null
}

/**
 * Registers single-key shortcuts for as long as the calling component is mounted.
 * Keys are `KeyboardEvent.key` values, e.g. `{ n: addCard, '/': focusSearch }`.
 *
 * Escape always fires. Any other key is ignored while the user is typing in a field,
 * holding Ctrl/Cmd/Alt, or while a modal dialog is open.
 */
export function useKeyboardShortcuts(shortcuts: Record<string, ShortcutHandler>) {
  // onKeyStroke adds a keydown listener and removes it when the component unmounts.
  onKeyStroke(Object.keys(shortcuts), (event) => {
    const handler = shortcuts[event.key]
    if (!handler) return

    if (event.key !== 'Escape') {
      if (event.ctrlKey || event.metaKey || event.altKey) return
      if (isTypingTarget(event.target) || isModalOpen()) return
    }

    // Without this, a shortcut that focuses an input would also type its own key into it.
    event.preventDefault()
    handler(event)
  })
}
