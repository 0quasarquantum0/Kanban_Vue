import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { installFakeServer, type FakeServer } from '@/__tests__/fakeServer'
import router from '../index'

let server: FakeServer

function seedIds(state: FakeServer['state']) {
  const board = Object.values(state.boards)[0]
  const card = Object.values(state.cards)[0]
  if (!board || !card) throw new Error('seed is empty')
  return { boardId: board.id, cardId: card.id }
}

describe('route guards', () => {
  beforeEach(async () => {
    server = installFakeServer()
    setActivePinia(createPinia())
    await router.push('/')
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('waits for the first load before judging whether the board exists', async () => {
    const { boardId } = seedIds(server.state)

    // Nothing has asked the store to load yet: the guard has to do it and wait.
    await router.push(`/boards/${boardId}`)

    expect(router.currentRoute.value.name).toBe('board')
    expect(server.requests).toEqual(['GET /api/state'])
  })

  it('redirects an unknown board to the board list', async () => {
    await router.push('/boards/not-a-board')

    expect(router.currentRoute.value.name).toBe('boards')
  })

  it('opens a card that is on the board and redirects one that is not', async () => {
    const { boardId, cardId } = seedIds(server.state)

    await router.push(`/boards/${boardId}/cards/${cardId}`)
    expect(router.currentRoute.value.name).toBe('card')

    // beforeEnter only runs when the route is entered, not when its params change.
    await router.push('/')
    await router.push(`/boards/${boardId}/cards/nope`)
    expect(router.currentRoute.value.fullPath).toBe(`/boards/${boardId}`)
  })

  it('lets the navigation through when the server cannot be reached', async () => {
    server.online = false

    await router.push('/boards/some-board')

    // Redirecting would lose the deep link; App.vue shows the error instead.
    expect(router.currentRoute.value.fullPath).toBe('/boards/some-board')
  })
})
