import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import InlineEdit from '../InlineEdit.vue'

function mountInlineEdit(modelValue = 'To do') {
  return mount(InlineEdit, {
    props: { modelValue, label: 'Column title' },
    // Without this, focus() calls and blur events would act on a detached element.
    attachTo: document.body,
  })
}

describe('InlineEdit', () => {
  it('shows the value as a button until clicked', async () => {
    const wrapper = mountInlineEdit()

    expect(wrapper.get('button').text()).toBe('To do')
    await wrapper.get('button').trigger('click')
    expect(wrapper.get('input').element.value).toBe('To do')

    wrapper.unmount()
  })

  it('emits update:modelValue with the trimmed text on Enter', async () => {
    const wrapper = mountInlineEdit()

    await wrapper.get('button').trigger('click')
    await wrapper.get('input').setValue('  Backlog  ')
    await wrapper.get('input').trigger('keydown', { key: 'Enter' })

    // This is the event the parent's v-model listens for.
    expect(wrapper.emitted('update:modelValue')).toEqual([['Backlog']])
    expect(wrapper.find('input').exists()).toBe(false)

    wrapper.unmount()
  })

  it('emits nothing on Escape, or when the text is empty or unchanged', async () => {
    const wrapper = mountInlineEdit()

    for (const [value, key] of [
      ['Backlog', 'Escape'],
      ['   ', 'Enter'],
      ['To do', 'Enter'],
    ] as const) {
      await wrapper.get('button').trigger('click')
      await wrapper.get('input').setValue(value)
      await wrapper.get('input').trigger('keydown', { key })
    }

    expect(wrapper.emitted('update:modelValue')).toBeUndefined()

    wrapper.unmount()
  })
})
