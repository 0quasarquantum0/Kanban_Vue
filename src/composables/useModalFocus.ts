import { onMounted, onUnmounted } from 'vue'
import type { ShallowRef } from 'vue'

const FOCUSABLE =
  'a[href], button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), summary'

/**
 * Focus management for a modal that is open from mount to unmount:
 * moves focus in, keeps Tab inside, and hands focus back at the end.
 *
 * Returns a keydown handler to bind with `@keydown.tab` on the dialog element.
 */
export function useModalFocus(dialog: Readonly<ShallowRef<HTMLElement | null>>) {
  // A composable runs during the component's setup, before anything in the modal exists,
  // so the focused element right now is whatever opened it.
  const opener = document.activeElement

  // Lifecycle hooks called inside a composable attach to the component that called it.
  // Focus goes to the field marked `data-autofocus`, or to the dialog itself.
  onMounted(() => {
    const target = dialog.value?.querySelector<HTMLElement>('[data-autofocus]') ?? dialog.value
    target?.focus()
  })

  // Runs however the modal goes away: Save, Esc, or the browser's Back button.
  onUnmounted(() => {
    if (opener instanceof HTMLElement && opener.isConnected) opener.focus()
  })

  /** Wraps Tab and Shift+Tab at either end of the dialog. */
  function trapFocus(event: KeyboardEvent) {
    const items = [...(dialog.value?.querySelectorAll<HTMLElement>(FOCUSABLE) ?? [])]
    const first = items[0]
    const last = items[items.length - 1]
    if (!first || !last) return

    const active = document.activeElement
    if (event.shiftKey && (active === first || active === dialog.value)) {
      event.preventDefault()
      last.focus()
    } else if (!event.shiftKey && active === last) {
      event.preventDefault()
      first.focus()
    }
  }

  return { trapFocus }
}
