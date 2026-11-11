<script setup lang="ts">
import { useId } from 'vue'
import { onKeyStroke } from '@vueuse/core'

// Setting this to false (from inside or from the parent) plays the leave animation.
const open = defineModel<boolean>('open', { required: true })

defineProps<{
  title: string
}>()

const emit = defineEmits<{
  /** Fired once the leave animation has finished. */
  closed: []
}>()

const titleId = useId()

onKeyStroke('Escape', () => {
  open.value = false
})
</script>

<template>
  <!-- Teleport renders this markup under <body>, outside the board's DOM and stacking context. -->
  <Teleport to="body">
    <!-- `appear` runs the enter animation on first render, since the modal mounts already open. -->
    <Transition name="modal" appear @after-leave="emit('closed')">
      <!-- mousedown, not click: a text-selection drag that ends on the backdrop shouldn't close. -->
      <div v-if="open" class="modal" @mousedown.self="open = false">
        <div class="modal__dialog" role="dialog" aria-modal="true" :aria-labelledby="titleId">
          <header class="modal__header">
            <h2 :id="titleId" class="modal__title">{{ title }}</h2>
            <button
              class="icon-button modal__close"
              type="button"
              aria-label="Close"
              @click="open = false"
            >
              ×
            </button>
          </header>
          <slot />
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal {
  position: fixed;
  inset: 0;
  z-index: 10;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 10vh var(--space-4) var(--space-4);
  overflow-y: auto;
  background: rgb(31 35 40 / 50%);
}

.modal__dialog {
  width: min(32rem, 100%);
  padding: var(--space-4);
  background: var(--color-surface);
  border-radius: var(--radius);
  box-shadow: 0 8px 24px rgb(31 35 40 / 20%);
}

.modal__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
}

.modal__title {
  margin: 0;
  font-size: 1.125rem;
}

.modal__close {
  font-size: 1.25rem;
}

.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}

.modal-enter-active .modal__dialog,
.modal-leave-active .modal__dialog {
  transition: transform 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .modal__dialog,
.modal-leave-to .modal__dialog {
  transform: translateY(0.5rem);
}

@media (prefers-reduced-motion: reduce) {
  .modal-enter-active,
  .modal-leave-active,
  .modal-enter-active .modal__dialog,
  .modal-leave-active .modal__dialog {
    transition: none;
  }
}
</style>
