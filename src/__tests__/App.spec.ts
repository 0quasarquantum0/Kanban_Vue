import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import App from '@/App.vue'
import router from '@/router'
import { installFakeServer, type FakeServer } from './fakeServer'

let server: FakeServer

function mountApp() {
  return mount(App, { global: { plugins: [createPinia(), router] } })
}

describe('App', () => {
  beforeEach(async () => {
    server = installFakeServer()
    await router.push('/')
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('shows a loading message until the state arrives, then the route', async () => {
    const wrapper = mountApp()
    expect(wrapper.text()).toContain('Loading your boards')

    await flushPromises()

    expect(wrapper.text()).not.toContain('Loading your boards')
    expect(wrapper.text()).toContain('Vue Kanban') // the seed board's title on the list view
  })

  it('shows an error with a retry button when the server is unreachable', async () => {
    server.online = false
    const wrapper = mountApp()

    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain("Can't load your boards")
    expect(wrapper.text()).toContain('Could not reach the server')
    expect(wrapper.find('button').text()).toBe('Try again')
  })
})
