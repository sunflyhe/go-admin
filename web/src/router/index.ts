// 路由:静态路由(登录/布局/403)+ 根据后端菜单动态注册的页面路由。
import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const views = import.meta.glob('../views/**/*.vue')

function viewComponent(component: string) {
  const loader = views[`../views/${component}.vue`]
  return loader
}

export const staticRoutes: RouteRecordRaw[] = [
  { path: '/login', name: 'login', component: () => import('../views/Login.vue'), meta: { public: true } },
  {
    path: '/',
    name: 'layout',
    component: () => import('../layout/AdminLayout.vue'),
    redirect: '/dashboard',
    children: [
      { path: 'dashboard', name: 'dashboard', component: () => import('../views/dashboard/index.vue'), meta: { title: '仪表盘' } },
      { path: '403', name: 'forbidden', component: () => import('../views/Forbidden.vue'), meta: { title: '无权限' } }
    ]
  }
]

export const router = createRouter({
  history: createWebHistory(),
  routes: staticRoutes
})

let dynamicForUserID: number | null = null
const dynamicRouteNames = new Set<string>()

function resetDynamicRoutes() {
  for (const name of dynamicRouteNames) router.removeRoute(name)
  dynamicRouteNames.clear()
  dynamicForUserID = null
}

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (to.meta.public) return true
  if (!auth.isLoggedIn) return { path: '/login', query: { redirect: to.fullPath } }
  try {
    const me = await auth.fetchMe()
    if (!me) return { path: '/login' }
    // 根据当前账号的菜单注册路由。账号切换时必须移除前一账号的路由，
    // 否则新用户会复用旧用户页面，或缺少自身应有页面。
    if (dynamicForUserID !== me.user.id) {
      resetDynamicRoutes()
      const addRoutes: RouteRecordRaw[] = []
      const walk = (nodes: NonNullable<typeof me.menus>) => {
        for (const n of nodes) {
          if (n.component && viewComponent(n.component)) {
            addRoutes.push({
              path: n.path,
              name: `menu-${n.id}`,
              component: viewComponent(n.component)!,
              meta: { title: n.name }
            })
          }
          if (n.children?.length) walk(n.children)
        }
      }
      walk(me.menus)
      for (const r of addRoutes) {
        router.addRoute('layout', r)
        dynamicRouteNames.add(String(r.name))
      }
      dynamicForUserID = me.user.id
      // 动态路由加入后重新导航一次
      return to.fullPath
    }
    return true
  } catch {
    return { path: '/login' }
  }
})
