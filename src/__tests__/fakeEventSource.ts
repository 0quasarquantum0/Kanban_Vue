import { vi } from 'vitest'

/**
 * A stand-in for the browser's EventSource (jsdom has none). Tests call `emitVersion` to play
 * the server announcing a new version, as /api/events does.
 */
export class FakeEventSource {
  static instances: FakeEventSource[] = []

  readonly url: string
  closed = false
  private listeners = new Map<string, ((event: MessageEvent<string>) => void)[]>()

  constructor(url: string) {
    this.url = url
    FakeEventSource.instances.push(this)
  }

  addEventListener(type: string, listener: (event: MessageEvent<string>) => void) {
    this.listeners.set(type, [...(this.listeners.get(type) ?? []), listener])
  }

  close() {
    this.closed = true
  }

  emitVersion(version: number) {
    const event = new MessageEvent('version', { data: JSON.stringify({ version }) })
    for (const listener of this.listeners.get('version') ?? []) listener(event)
  }
}

/** Installs the fake as the global EventSource and forgets earlier instances. */
export function installFakeEventSource(): void {
  FakeEventSource.instances = []
  vi.stubGlobal('EventSource', FakeEventSource)
}
