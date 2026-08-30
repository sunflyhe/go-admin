import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('../views/Home.vue')
    },
    // 兜底:未知路径(如 stale 链接)回首页,避免白屏
    { path: '/:pathMatch(.*)*', redirect: '/' }
  ]
})

export default router
