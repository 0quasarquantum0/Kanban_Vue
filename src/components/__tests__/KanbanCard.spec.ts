import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import type { Card } from '@/types'
import KanbanCard from '../KanbanCard.vue'
import { createTestRouter } from './testRouter'

function makeCard(overrides: Partial<Card> = {}): Card {
  return {
    id: 'c1',
    title: 'Fix the login bug',
    description: '',
    labels: ['bug', 'ui'],
    dueDate: null,
    createdAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

async function mountCard(card: Card) {
  const router = await createTestRouter('/boards/b1?label=bug')
  // global.plugins installs the router the same way app.use(router) does in main.ts.
  return mount(KanbanCard, { props: { card }, global: { plugins: [router] } })
}

describe('KanbanCard', () => {
  it('renders its title and labels', async () => {
    const wrapper = await mountCard(makeCard())

    expect(wrapper.get('.card__title').text()).toBe('Fix the login bug')
    expect(wrapper.findAll('.card__label').map((label) => label.text())).toEqual(['bug', 'ui'])
  })

  it('links the title to the card route, keeping the current query', async () => {
    const wrapper = await mountCard(makeCard())

    expect(wrapper.get('a.card__link').attributes('href')).toBe('/boards/b1/cards/c1?label=bug')
  })

  it('renders no label list or footer when the card has neither labels nor due date', async () => {
    const wrapper = await mountCard(makeCard({ labels: [] }))

    expect(wrapper.find('.card__labels').exists()).toBe(false)
    expect(wrapper.find('.card__meta').exists()).toBe(false)
  })

  it('emits delete when the × button is clicked', async () => {
    const wrapper = await mountCard(makeCard())

    await wrapper.get('button[aria-label^="Delete card"]').trigger('click')

    expect(wrapper.emitted('delete')).toHaveLength(1)
  })
})
