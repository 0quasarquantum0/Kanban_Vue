import { createRouter, createWebHistory } from 'vue-router'
import type { RouteLocationNormalized, RouteLocationRaw } from 'vue-router'
import CardEditor from '@/components/CardEditor.vue'
import { useKanbanStore } from '@/stores/kanban'
import BoardDetailView from '@/views/BoardDetailView.vue'
import BoardsListView from '@/views/BoardsListView.vue'

// Guards return `true` to continue, or a location to redirect to.
// The store is looked up inside the guard: Pinia isn't installed yet when this module loads.
//
// The data now arrives from the server, so the guards are async: they wait for the first load
// before judging whether a board or card exists. If that load failed there is nothing to judge
// against, so they let the navigation through (App.vue shows the error instead of the route).

async function requireBoard(to: RouteLocationNormalized): Promise<true | RouteLocationRaw> {
  const store = useKanbanStore()
  await store.ensureLoaded()
  if (store.status === 'error') return true
  const boardId = String(to.params.boardId)
  return store.state.boards[boardId] ? true : { name: 'boards' }
}

async function requireCard(to: RouteLocationNormalized): Promise<true | RouteLocationRaw> {
  const store = useKanbanStore()
  await store.ensureLoaded()
  if (store.status === 'error') return true
  const boardId = String(to.params.boardId)
  const cardId = String(to.params.cardId)
  const isOnBoard = store.columnsForBoard(boardId).some((column) => column.cardIds.includes(cardId))
  return isOnBoard ? true : { name: 'board', params: { boardId } }
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'boards',
      component: BoardsListView,
    },
    {
      path: '/boards/:boardId',
      name: 'board',
      component: BoardDetailView,
      // Pass the param to the component as a prop, so it doesn't need useRoute for it.
      // A function, not `true`: on the card route `true` would also pass `cardId` here.
      props: (route) => ({ boardId: String(route.params.boardId) }),
      beforeEnter: requireBoard,
      children: [
        {
          // Renders inside BoardDetailView's own <RouterView>, over the board.
          path: 'cards/:cardId',
          name: 'card',
          component: CardEditor,
          // `true` passes every param as a prop: here both boardId and cardId.
          props: true,
          beforeEnter: requireCard,
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: { name: 'boards' },
    },
  ],
})

export default router
