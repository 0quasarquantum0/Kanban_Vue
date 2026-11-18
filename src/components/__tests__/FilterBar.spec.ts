import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import FilterBar from '../FilterBar.vue'
import { createTestRouter } from './testRouter'

async function mountFilterBar(initialPath = '/boards/b1') {
  const router = await createTestRouter(initialPath)
  const wrapper = mount(FilterBar, {
    props: { labels: ['bug', 'ui'] },
    global: { plugins: [router] },
  })
  return { wrapper, router }
}

describe('FilterBar', () => {
  // Fake timers let the test skip over the 300ms search debounce.
  beforeEach(() => {
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('writes the search text to ?q= once typing pauses', async () => {
    const { wrapper, router } = await mountFilterBar()

    await wrapper.get('input[type="search"]').setValue('login')
    expect(router.currentRoute.value.query.q).toBeUndefined()

    await vi.advanceTimersByTimeAsync(300)
    await flushPromises() // router.replace resolves asynchronously

    expect(router.currentRoute.value.query.q).toBe('login')
  })

  it('toggles labels in ?label=', async () => {
    const { wrapper, router } = await mountFilterBar()
    const button = (label: string) =>
      wrapper.findAll('.filters__label').find((item) => item.text() === label)

    await button('bug')?.trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query.label).toEqual(['bug'])

    await button('ui')?.trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query.label).toEqual(['bug', 'ui'])

    await button('bug')?.trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query.label).toEqual(['ui'])
  })

  it('reads its starting state from the URL, and clears it', async () => {
    const { wrapper, router } = await mountFilterBar('/boards/b1?q=login&label=bug')

    const input = wrapper.get<HTMLInputElement>('input[type="search"]')
    expect(input.element.value).toBe('login')
    expect(wrapper.get('.filters__label--active').text()).toBe('bug')

    await wrapper.get('.filters__clear').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.query).toEqual({})
    expect(input.element.value).toBe('')
  })
})
