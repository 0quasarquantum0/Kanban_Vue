<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useKanbanStore } from '@/stores/kanban'
import type { ID } from '@/types'
import BaseModal from './BaseModal.vue'

// Both come from the route params (`props: true` on the route).
const props = defineProps<{
  boardId: ID
  cardId: ID
}>()

const store = useKanbanStore()
const router = useRouter()
const route = useRoute()

const isOpen = ref(true)

// A local copy of the card: edits only reach the store on Save. `reactive` suits a group
// of form fields, since there's no `.value` and the object is never reassigned.
const card = store.state.cards[props.cardId]
const draft = reactive({
  title: card?.title ?? '',
  description: card?.description ?? '',
  labels: card?.labels.join(', ') ?? '',
  dueDate: card?.dueDate ?? '',
})

function parseLabels(text: string): string[] {
  const labels = text.split(',').map((label) => label.trim().toLowerCase())
  return [...new Set(labels.filter(Boolean))]
}

function save() {
  store.updateCard(props.cardId, {
    title: draft.title.trim(),
    description: draft.description.trim(),
    labels: parseLabels(draft.labels),
    dueDate: draft.dueDate || null,
  })
  isOpen.value = false
}

function remove() {
  store.deleteCard(props.cardId)
  isOpen.value = false
}

// Runs after the modal's leave animation. Closing is a navigation back to the board route.
function goToBoard() {
  router.push({ name: 'board', params: { boardId: props.boardId }, query: route.query })
}
</script>

<template>
  <BaseModal v-model:open="isOpen" title="Edit card" @closed="goToBoard">
    <form class="editor" @submit.prevent="save">
      <label class="editor__field">
        <span class="editor__label">Title</span>
        <input
          v-model.trim="draft.title"
          class="editor__input"
          type="text"
          required
          data-autofocus
        />
      </label>

      <label class="editor__field">
        <span class="editor__label">Description</span>
        <textarea v-model="draft.description" class="editor__input" rows="4" />
      </label>

      <label class="editor__field">
        <span class="editor__label">Labels <small>(comma separated)</small></span>
        <input v-model="draft.labels" class="editor__input" type="text" placeholder="bug, ui" />
      </label>

      <label class="editor__field">
        <span class="editor__label">Due date</span>
        <input v-model="draft.dueDate" class="editor__input editor__input--date" type="date" />
      </label>

      <footer class="editor__actions">
        <button class="button button--danger-outline editor__delete" type="button" @click="remove">
          Delete card
        </button>
        <button class="button" type="button" @click="isOpen = false">Cancel</button>
        <button class="button button--primary" type="submit">Save</button>
      </footer>
    </form>
  </BaseModal>
</template>

<style scoped>
.editor {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.editor__field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.editor__label {
  font-size: 0.8125rem;
  font-weight: 600;
}

.editor__label small {
  font-weight: 400;
  color: var(--color-text-muted);
}

.editor__input {
  padding: var(--space-2);
  font: inherit;
  font-size: 0.9375rem;
  color: inherit;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  resize: vertical;
}

.editor__input:focus {
  border-color: var(--color-accent);
  outline: 2px solid var(--color-label-bg);
}

.editor__input--date {
  align-self: flex-start;
}

.editor__actions {
  display: flex;
  gap: var(--space-2);
  margin-top: var(--space-2);
}

.editor__delete {
  margin-right: auto;
}
</style>
