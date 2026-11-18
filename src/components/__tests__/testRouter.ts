import { defineComponent } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'

const Empty = defineComponent({ template: '<div />' })

/**
 * A real router with the app's board and card routes, but empty components and in-memory
 * history, so components that use RouterLink, useRoute or useRouter can be mounted alone.
 */
export async function createTestRouter(initialPath = '/boards/b1') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'boards', component: Empty },
      {
        path: '/boards/:boardId',
        name: 'board',
        component: Empty,
        children: [{ path: 'cards/:cardId', name: 'card', component: Empty }],
      },
    ],
  })
  await router.push(initialPath)
  await router.isReady()
  return router
}
