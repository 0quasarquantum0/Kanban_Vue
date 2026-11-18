import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import App from '@/App.vue'
import router from '@/router'
import { installFakeServer, type FakeServer } from './fakeServer'

let server: FakeServer

async function mountAppAt(path: string) {
  // The route guards look the store up through the active Pinia, so it must be this test's.
  const pinia = createPinia()
  setActivePinia(pinia)
  await router.push(path)
  const wrapper = mount(App, { global: { plugins: [pinia, router] }, attachTo: document.body })
  await flushPromises()
  return wrapper
}

/** Types a new text into the InlineEdit that currently shows `current`, and presses Enter. */
async function rename(wrapper: VueWrapper, current: string, next: string) {
  const button = wrapper.findAll('.inline-edit--text').find((item) => item.text() === current)
  if (!button) throw new Error(`no editable text "${current}"`)
  await button.trigger('click')
  const input = wrapper.get('input.inline-edit--input')
  await input.setValue(next)
  await input.trigger('keydown', { key: 'Enter' })
  await flushPromises()
}

describe('renaming goes through the store actions, so the server hears about it', () => {
  beforeEach(() => {
    server = installFakeServer()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    document.body.innerHTML = ''
  })

  it('renames a board from the board list', async () => {
    const wrapper = await mountAppAt('/')
    const boardId = Object.keys(server.state.boards)[0] ?? ''

    await rename(wrapper, 'Vue Kanban', 'Renamed board')
    await vi.waitFor(() => expect(server.requests).toContain(`PATCH /api/boards/${boardId}`))

    expect(server.state.boards[boardId]?.title).toBe('Renamed board')
    wrapper.unmount()
  })

  it('renames a column on the board', async () => {
    const boardId = Object.keys(server.state.boards)[0] ?? ''
    const columnId = Object.values(server.state.columns).find((c) => c.title === 'To do')?.id ?? ''
    const wrapper = await mountAppAt(`/boards/${boardId}`)

    await rename(wrapper, 'To do', 'Backlog')
    await vi.waitFor(() => expect(server.requests).toContain(`PATCH /api/columns/${columnId}`))

    expect(server.state.columns[columnId]?.title).toBe('Backlog')
    wrapper.unmount()
  })
})
