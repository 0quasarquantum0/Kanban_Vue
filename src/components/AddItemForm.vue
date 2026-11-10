<script setup lang="ts">
import { nextTick, ref, useTemplateRef } from 'vue'

defineProps<{
  /** Button text, e.g. "Add card". */
  label: string
  placeholder?: string
}>()

const emit = defineEmits<{
  add: [title: string]
}>()

const isOpen = ref(false)
const title = ref('')
const input = useTemplateRef('input')

async function open() {
  isOpen.value = true
  await nextTick()
  input.value?.focus()
}

function close() {
  isOpen.value = false
  title.value = ''
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
  if (!title.value.trim()) close()
}
</script>

<template>
  <div v-if="isOpen" class="add-item">
    <input
      ref="input"
      v-model="title"
      class="add-item__input"
      type="text"
      :placeholder="placeholder"
      :aria-label="placeholder ?? label"
      @keyup.enter="submit"
      @keyup.esc="close"
      @blur="closeIfEmpty"
    />
    <button class="add-item__submit" type="button" @click="submit">Add</button>
  </div>
  <button v-else class="add-item__open" type="button" @click="open">+ {{ label }}</button>
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
  color: #fff;
  background: var(--color-accent);
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
