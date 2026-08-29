<template>
  <div class="dashboard">
    <!-- 欢迎横幅 -->
    <div class="hero">
      <div class="hero-text">
        <h2>{{ greeting }},{{ displayName }}</h2>
        <p>{{ todayText }} · 欢迎使用 Go Admin 管理后台</p>
        <div v-if="auth.user?.roles?.length" class="hero-roles">
          <span v-for="r in auth.user.roles" :key="r" class="role-chip">{{ r }}</span>
        </div>
      </div>
      <el-icon class="hero-icon" :size="72"><Platform /></el-icon>
    </div>

    <!-- 常用功能:来自服务端下发的菜单 -->
    <el-card class="section">
      <template #header>
        <span class="section-title">常用功能</span>
      </template>
      <div v-if="pages.length" class="quick-grid">
        <div v-for="p in pages" :key="p.id" class="quick-item" @click="router.push(p.path)">
          <span class="quick-icon">
            <el-icon :size="20"><component :is="p.icon || 'Menu'" /></el-icon>
          </span>
          <span class="quick-name">{{ p.name }}</span>
        </div>
      </div>
      <el-empty v-else description="暂无可访问的菜单" :image-size="80" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Platform } from '@element-plus/icons-vue'
import { useAuthStore } from '../../stores/auth'
import type { MenuNode } from '../../api'

const auth = useAuthStore()
const router = useRouter()
const now = new Date()

const displayName = computed(() => auth.user?.nickname || auth.user?.username || '')
const greeting = computed(() => {
  const h = now.getHours()
  if (h < 6) return '夜深了'
  if (h < 12) return '早上好'
  if (h < 14) return '中午好'
  if (h < 18) return '下午好'
  return '晚上好'
})
const todayText = computed(() =>
  `${now.getFullYear()} 年 ${now.getMonth() + 1} 月 ${now.getDate()} 日 · 星期${'日一二三四五六'[now.getDay()]}`,
)

// 服务端菜单中的页面节点(type=2),作为工作台快捷入口;权限始终以服务端下发为准
const pages = ref<MenuNode[]>([])
onMounted(async () => {
  const me = await auth.fetchMe()
  const result: MenuNode[] = []
  const walk = (nodes: MenuNode[]) => {
    for (const n of nodes) {
      if (n.type === 2) result.push(n)
      if (n.children?.length) walk(n.children)
    }
  }
  walk(me?.menus ?? [])
  pages.value = result
})
</script>

<style scoped>
.dashboard {
  width: 100%;
}

.hero {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 26px 30px;
  margin-bottom: 14px;
  overflow: hidden;
  color: #fff;
  background: linear-gradient(120deg, #2563eb, #4f8bff 55%, #7ca1f3);
  border-radius: 12px;
  box-shadow: 0 10px 30px rgb(37 99 235 / 25%);
}

.hero h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
}

.hero p {
  margin: 8px 0 0;
  font-size: 13px;
  opacity: 0.85;
}

.hero-roles {
  display: flex;
  gap: 8px;
  margin-top: 12px;
}

.role-chip {
  padding: 2px 10px;
  font-size: 12px;
  background: rgb(255 255 255 / 20%);
  border-radius: 10px;
}

.hero-icon {
  opacity: 0.25;
}

.section {
  border-radius: 10px;
}

.section-title {
  font-size: 15px;
  font-weight: 600;
  color: #1d2129;
}

.quick-grid {
  /* auto-fit:条目不足一行时平摊整行宽度,避免右侧留空 */
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 10px;
}

.quick-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 18px 10px 14px;
  cursor: pointer;
  border: 1px solid transparent;
  border-radius: 10px;
  transition: background-color 0.2s, border-color 0.2s, transform 0.2s;
}

.quick-item:hover {
  background: var(--el-color-primary-light-9);
  border-color: var(--el-color-primary-light-8);
  transform: translateY(-2px);
}

.quick-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 42px;
  height: 42px;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-radius: 12px;
}

.quick-item:hover .quick-icon {
  color: #fff;
  background: var(--el-color-primary);
}

.quick-name {
  font-size: 13px;
  color: #4e5969;
}
</style>
