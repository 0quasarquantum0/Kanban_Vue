import { ref } from 'vue'
import type { Ref } from 'vue'
import { watchDebounced } from '@vueuse/core'
import { ApiError, getState, putState } from '@/api'
import type { KanbanState } from '@/types'

/** `idle` until the first load starts; `error` only when the first load failed. */
export type SyncStatus = 'idle' | 'loading' | 'ready' | 'error'

export const SAVE_DELAY_MS = 300

function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

/**
 * Keeps `state` in step with the Go server: loads it once, then saves every change with a
 * debounced deep watch (the same idea as `useStorage` in phase 5, with a server behind it).
 *
 * The server tags each state with a version. A save carries the version this tab last saw; if
 * another tab saved in between the server answers 409, and this tab reloads and tells the user.
 */
export function useServerSync(state: Ref<KanbanState>) {
  const status = ref<SyncStatus>('idle')
  const loadError = ref<string | null>(null)
  /** A message for the user about something that went wrong after the first load. */
  const notice = ref<string | null>(null)
  const version = ref(0)

  // The JSON the server is known to hold. Comparing against it stops the watch from saving
  // state that was only just loaded, and makes a save with no real change a no-op.
  let syncedJson = JSON.stringify(state.value)
  let firstLoad: Promise<void> | null = null
  let isSaving = false
  let saveAgain = false

  async function load(): Promise<void> {
    if (status.value !== 'ready') status.value = 'loading'
    try {
      const snapshot = await getState()
      state.value = snapshot.state
      version.value = snapshot.version
      syncedJson = JSON.stringify(snapshot.state)
      loadError.value = null
      status.value = 'ready'
    } catch (error) {
      if (status.value === 'ready') {
        notice.value = `Couldn't refresh from the server: ${messageOf(error)}`
      } else {
        loadError.value = messageOf(error)
        status.value = 'error'
      }
    }
  }

  /** Starts the first load if it has not started, and resolves when that load has finished. */
  function ensureLoaded(): Promise<void> {
    firstLoad ??= load()
    return firstLoad
  }

  /** Sends the state to the server now, if it differs from what the server has. */
  async function saveNow(): Promise<void> {
    if (status.value !== 'ready') return
    if (isSaving) {
      saveAgain = true // the running save will go round again once it finishes
      return
    }

    isSaving = true
    try {
      do {
        saveAgain = false
        const json = JSON.stringify(state.value)
        if (json === syncedJson) continue
        version.value = await putState(version.value, state.value)
        syncedJson = json
        notice.value = null
      } while (saveAgain)
    } catch (error) {
      if (error instanceof ApiError && error.status === 409) {
        await load()
        notice.value =
          'Your last change conflicted with an edit made in another tab or window. ' +
          'The board was reloaded, so that change was not saved.'
      } else {
        notice.value = `Couldn't save your changes: ${messageOf(error)}`
      }
    } finally {
      isSaving = false
    }
  }

  watchDebounced(state, saveNow, { deep: true, debounce: SAVE_DELAY_MS, maxWait: 2000 })

  function dismissNotice() {
    notice.value = null
  }

  return { status, loadError, notice, version, load, ensureLoaded, saveNow, dismissNotice }
}
