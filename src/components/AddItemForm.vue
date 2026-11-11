<script setup lang="ts">
import { nextTick, ref, useTemplateRef, watch } from 'vue'
import type { Directive } from 'vue'

defineProps<{
  /** Button text, e.g. "Add card". */
  label: string
  placeholder?: string
}>()

const emit = defineEmits<{
  add: [title: string]
}>()

// Works as local state when the parent doesn't bind it, and as a two-way binding when it
// does. BoardView binds it so the `n` shortcut can open the first column's form.
const isOpen = defineModel<boolean>('open', { default: false })
const title = ref('')
const input = useTemplateRef('input')
const openButton = useTemplateRef('openButton')

// A local custom directive, used as `v-focus`. `mounted` runs when the element is
// inserted, so the input takes focus however the form was opened.
const vFocus: Directive<HTMLElement> = { mounted: (el) => el.focus() }

watch(isOpen, (open) => {
  if (!open) title.value = ''
})

// Esc: close and hand focus back, so a keyboard user stays where they were.
async function cancel() {
  isOpen.value = false
  await nextTick()
  openButton.value?.focus()
}

// Stays open after submitting so several items can be added in a row.
function submit() {
  const value = title.value.trim()
  if (!value) return
  emit('add', value)
  title.value = ''
  input.value?.focus()
}

function closeIfEmpty() {
  if (!title.value.trim()) isOpen.value = false
}
</script>

<template>
  <div v-if="isOpen" class="add-item">
    <input
      ref="input"
      v-model="title"
      v-focus
      class="add-item__input"
      type="text"
      :placeholder="placeholder"
      :aria-label="placeholder ?? label"
      @keyup.enter="submit"
      @keyup.esc="cancel"
      @blur="closeIfEmpty"
    />
    <button class="add-item__submit" type="button" @click="submit">Add</button>
  </div>
  <button v-else ref="openButton" class="add-item__open" type="button" @click="isOpen = true">
    + {{ label }}
  </button>
</template>

<style scoped>
.add-item {
  display: flex;
  gap: var(--space-2);
}

.add-item__input {
  flex: 1;
  min-width: 0;
  padding: var(--space-1) var(--space-2);
  font: inherit;
  font-size: 0.875rem;
  color: inherit;
  background: var(--color-surface);
  border: 1px solid var(--color-accent);
  border-radius: var(--radius);
  outline: none;
}

.add-item__submit,
.add-item__open {
  padding: var(--space-1) var(--space-2);
  font: inherit;
  font-size: 0.875rem;
  border: none;
  border-radius: var(--radius);
  cursor: pointer;
}

.add-item__submit {
  color: var(--color-on-fill);
  background: var(--color-accent-fill);
}

.add-item__open {
  width: 100%;
  text-align: left;
  color: var(--color-text-muted);
  background: none;
}

.add-item__open:hover {
  color: var(--color-text);
  background: var(--color-hover);
}
</style>
