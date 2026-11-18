import { ref } from 'vue'
import type { Ref } from 'vue'
import { ApiError, getState } from '@/api'
import type { Snapshot, Written } from '@/api'
import type { ID, KanbanState } from '@/types'

/** `idle` until the first load starts; `error` only when the first load failed. */
export type SyncStatus = 'idle' | 'loading' | 'ready' | 'error'

function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

/**
 * Keeps `state` in step with the Go server.
 *
 * Reading: the state is loaded once, and reloaded when this tab learns it is behind.
 *
 * Writing: every store action changes `state` first (so the screen updates at once) and then
 * calls `enqueue` with the matching request. Requests run one at a time, in the order they were
 * made, because the server needs `addCard` before it can `updateCard` that card. If one fails,
 * the requests queued behind it are dropped and the state is reloaded from the server, which
 * undoes the optimistic changes that never made it there.
 */
export function useServerSync(state: Ref<KanbanState>) {
  const status = ref<SyncStatus>('idle')
  const loadError = ref<string | null>(null)
  /** A message for the user about something that went wrong after the first load. */
  const notice = ref<string | null>(null)
  /** The server version this tab has caught up with. */
  const version = ref(0)

  let firstLoad: Promise<void> | null = null
  let tail: Promise<void> = Promise.resolve() // the end of the request queue
  let pending = 0 // requests queued or running
  let generation = 0 // raised on every failure, so requests queued before it can be skipped

  // --- reading ---

  function apply(snapshot: Snapshot): void {
    state.value = snapshot.state
    version.value = snapshot.version
    loadError.value = null
    status.value = 'ready'
  }

  /**
   * Loads the server's state into the store. It waits for queued requests to finish first and,
   * if another change was queued while the response was on its way, throws the response away
   * and asks again: replacing the state then would wipe out a change the server has not seen.
   */
  async function load(): Promise<void> {
    if (status.value !== 'ready') status.value = 'loading'
    try {
      for (;;) {
        await settled()
        const snapshot = await getState()
        if (pending > 0) continue
        apply(snapshot)
        return
      }
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

  /** Reloads if the server has moved past `target` and this tab has not caught up. */
  async function refreshIfBehind(target: number): Promise<void> {
    await settled()
    if (version.value >= target) return
    await load()
  }

  // --- writing ---

  /** Resolves once every request queued so far has finished (or been skipped). */
  async function settled(): Promise<void> {
    let end: Promise<void>
    do {
      end = tail
      await end
    } while (end !== tail)
  }

  /** What the user is told when the request behind a change failed. */
  function describeFailure(error: unknown): string {
    if (error instanceof ApiError && error.status === 409) {
      return (
        'Your last change conflicted with an edit made in another tab or window. ' +
        'The board was reloaded, so that change was not saved.'
      )
    }
    if (error instanceof ApiError && error.status === 404) {
      return (
        'Something you changed no longer exists; it was probably deleted in another tab. ' +
        'The board was reloaded.'
      )
    }
    return `Couldn't save your change: ${messageOf(error)} The board was reloaded.`
  }

  async function recover(error: unknown): Promise<void> {
    generation += 1 // everything queued before this moment was built on state that is now wrong
    const message = describeFailure(error)
    try {
      apply(await getState())
      notice.value = message
    } catch (reloadError) {
      // The server cannot be reached at all, so the screen keeps the unsaved changes.
      notice.value = `Couldn't save your change: ${messageOf(reloadError)}`
    }
  }

  /** The reply to a write told us the version it produced. */
  function accept(written: Written, expectedId: ID | undefined): void {
    const gap = written.version > version.value + 1
    version.value = Math.max(version.value, written.version)
    const idDiffers = written.id !== undefined && written.id !== expectedId
    // Either another client wrote in between, or the server used a different id than ours.
    if (gap || idDiffers) void load()
  }

  /**
   * Queues a request. `expectedId` is the id the screen already uses for a thing that the
   * request creates; if the server picked another one, the state is reloaded.
   */
  function enqueue(request: () => Promise<Written>, expectedId?: ID): void {
    const queuedAt = generation
    pending += 1
    tail = tail.then(async () => {
      try {
        if (queuedAt !== generation) return // an earlier request failed; this change was undone
        accept(await request(), expectedId)
      } catch (error) {
        await recover(error)
      } finally {
        pending -= 1
      }
    })
  }

  function dismissNotice() {
    notice.value = null
  }

  return {
    status,
    loadError,
    notice,
    version,
    load,
    ensureLoaded,
    refreshIfBehind,
    settled,
    enqueue,
    dismissNotice,
  }
}
