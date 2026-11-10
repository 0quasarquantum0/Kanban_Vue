<script setup lang="ts">
import { nextTick, ref, useTemplateRef } from 'vue'

// defineModel declares a `modelValue` prop and an `update:modelValue` event in one line.
// Assigning to `model.value` emits the event; the parent's v-model does the rest.
const model = defineModel<string>({ required: true })

defineProps<{
  /** Accessible name for the input, e.g. "Column title". */
  label: string
}>()

const isEditing = ref(false)
const draft = ref('')
const input = useTemplateRef('input')

async function startEditing() {
  draft.value = model.value
  isEditing.value = true
  // The input only exists after Vue has patched the DOM.
  await nextTick()
  input.value?.select()
}

function commit() {
  if (!isEditing.value) return
  const value = draft.value.trim()
  if (value && value !== model.value) model.value = value
  isEditing.value = false
}

function cancel() {
  isEditing.value = false
}
</script>

<template>
  <input
    v-if="isEditing"
    ref="input"
    v-model="draft"
    class="inline-edit inline-edit--input"
    type="text"
    :aria-label="label"
    @keyup.enter="commit"
    @keyup.esc="cancel"
    @blur="commit"
  />
  <button
    v-else
    class="inline-edit inline-edit--text"
    type="button"
    :aria-label="`${label}: ${model}. Click to rename`"
    @click="startEditing"
  >
    {{ model }}
  </button>
</template>

<style scoped>
/* Both states inherit the surrounding heading's typography so the swap doesn't shift layout. */
.inline-edit {
  width: 100%;
  margin: 0;
  padding: 0 var(--space-1);
  font: inherit;
  letter-spacing: inherit;
  color: inherit;
  border: 1px solid transparent;
  border-radius: var(--radius);
}

.inline-edit--text {
  text-align: left;
  text-transform: inherit;
  overflow-wrap: anywhere;
  background: none;
  cursor: text;
}

.inline-edit--text:hover {
  background: var(--color-hover);
}

.inline-edit--input {
  background: var(--color-surface);
  border-color: var(--color-accent);
  outline: none;
}
</style>
