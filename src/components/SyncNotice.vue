<script setup lang="ts">
defineProps<{
  message: string
}>()

const emit = defineEmits<{
  dismiss: []
}>()
</script>

<template>
  <div class="notice" role="alert">
    <p class="notice__message">{{ message }}</p>
    <button class="button" type="button" @click="emit('dismiss')">Dismiss</button>
  </div>
</template>

<style scoped>
.notice {
  position: fixed;
  right: var(--space-4);
  bottom: var(--space-4);
  z-index: 30;
  display: flex;
  align-items: center;
  gap: var(--space-3);
  max-width: min(32rem, calc(100vw - 2 * var(--space-4)));
  padding: var(--space-2) var(--space-3);
  background: var(--color-surface);
  border: 1px solid var(--color-danger);
  border-radius: var(--radius);
  box-shadow: var(--shadow-card);
}

.notice__message {
  margin: 0;
  font-size: 0.875rem;
}
</style>
