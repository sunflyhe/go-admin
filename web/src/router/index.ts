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
      { path: '403', name: 'forbidden', component: () => import('../views/Forbidden.vue'), meta: { title: '无权限' } },
      // 非菜单功能页:不出现在侧边栏,由列表页按钮跳转进入
      { path: 'article/article/edit', name: 'article-edit', component: () => import('../views/article/article/edit.vue'), meta: { title: '编辑文章' } }
    ]
  },
  // 兜底:未注册路径。不能用 redirect(redirect 在守卫前执行,会把刷新时的原始 URL
  // 替换成 /dashboard,导致动态路由永远无法按原路径恢复);改为命名空路由,
  // 由 beforeEach 注册完动态路由后重试原路径,仍未匹配(真正的未知路径)再回仪表盘。
  // 该记录不会被真正渲染:公开路径直接放行,其余路径守卫总会重定向。
  { path: '/:pathMatch(.*)*', name: 'not-found', component: () => import('../views/Forbidden.vue') }
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
      // 动态路由加入后重新导航一次;to.fullPath 保留刷新时的原始地址
      return to.fullPath
    }
    // 路由已按当前账号注册完毕仍未匹配 → 真正的未知路径,回仪表盘避免空白
    if (to.name === 'not-found') return { path: '/dashboard' }
    return true
  } catch {
    return { path: '/login' }
  }
})
