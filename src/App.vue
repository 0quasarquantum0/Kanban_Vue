<script setup lang="ts">
import { RouterView } from 'vue-router'
import { provideTheme } from '@/composables/useTheme'

// Makes the theme available to every component below, however deep.
provideTheme()
</script>

<template>
  <RouterView />
</template>
