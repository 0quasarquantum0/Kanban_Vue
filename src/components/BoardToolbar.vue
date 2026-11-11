<script setup lang="ts">
import { ref, useTemplateRef } from 'vue'
import { useKanbanStore } from '@/stores/kanban'

const store = useKanbanStore()

const fileInput = useTemplateRef('fileInput')
const importError = ref<string | null>(null)
const isConfirmingReset = ref(false)

function exportJson() {
  const json = JSON.stringify(store.state, null, 2)
  const url = URL.createObjectURL(new Blob([json], { type: 'application/json' }))
  const link = document.createElement('a')
  link.href = url
  link.download = 'kanban-board.json'
  link.click()
  URL.revokeObjectURL(url)
}

async function importJson() {
  const input = fileInput.value
  const file = input?.files?.[0]
  if (!input || !file) return

  importError.value = null
  try {
    const data: unknown = JSON.parse(await file.text())
    if (!store.importState(data)) {
      importError.value = `"${file.name}" isn't a Kanban export, so nothing was changed.`
    }
  } catch {
    importError.value = `"${file.name}" isn't valid JSON, so nothing was changed.`
  }
  // Clear the input so choosing the same file again still fires `change`.
  input.value = ''
}

function reset() {
  store.resetToDemo()
  isConfirmingReset.value = false
  importError.value = null
}
</script>

<template>
  <div class="toolbar">
    <div v-if="isConfirmingReset" class="toolbar__group" role="group" aria-label="Confirm reset">
      <span class="toolbar__question">Replace everything with the demo data?</span>
      <button class="toolbar__button toolbar__button--danger" type="button" @click="reset">
        Yes, reset
      </button>
      <button class="toolbar__button" type="button" @click="isConfirmingReset = false">
        Cancel
      </button>
    </div>

    <div v-else class="toolbar__group">
      <button class="toolbar__button" type="button" @click="exportJson">Export JSON</button>
      <button class="toolbar__button" type="button" @click="fileInput?.click()">Import JSON</button>
      <button class="toolbar__button" type="button" @click="isConfirmingReset = true">
        Reset to demo data
      </button>
      <input
        ref="fileInput"
        type="file"
        accept="application/json,.json"
        hidden
        @change="importJson"
      />
    </div>

    <p v-if="importError" class="toolbar__error" role="alert">{{ importError }}</p>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: var(--space-1);
}

.toolbar__group {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
}

.toolbar__question {
  font-size: 0.875rem;
}

.toolbar__button {
  padding: var(--space-1) var(--space-3);
  font: inherit;
  font-size: 0.875rem;
  color: inherit;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  cursor: pointer;
}

.toolbar__button:hover {
  background: var(--color-hover);
}

.toolbar__button--danger {
  color: #fff;
  background: var(--color-danger);
  border-color: var(--color-danger);
}

.toolbar__button--danger:hover {
  background: var(--color-danger);
  filter: brightness(0.9);
}

.toolbar__error {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--color-danger);
}
</style>
