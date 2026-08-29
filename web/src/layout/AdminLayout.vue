<template>
  <el-container class="layout">
    <el-aside :width="collapsed ? '64px' : '220px'" class="aside">
      <div class="logo">
        <div class="logo-mark">
          <el-icon :size="18"><Platform /></el-icon>
        </div>
        <span v-show="!collapsed" class="logo-title">Go Admin</span>
      </div>
      <el-menu
        :default-active="$route.path"
        router
        :collapse="collapsed"
        :collapse-transition="false"
        class="menu admin-menu"
      >
        <template v-for="node in menus" :key="node.id">
          <AdminMenuItem :node="node" />
        </template>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <div class="header-left">
          <span class="collapse-btn" :title="collapsed ? '展开菜单' : '收起菜单'" @click="collapsed = !collapsed">
            <el-icon :size="17"><Expand v-if="collapsed" /><Fold v-else /></el-icon>
          </span>
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/dashboard' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item v-if="$route.meta.title">{{ $route.meta.title }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <el-dropdown @command="onCommand">
          <span class="user">
            <img v-if="auth.user?.avatar" :src="auth.user.avatar" :alt="auth.user?.nickname || '头像'" class="avatar avatar--img" />
            <span v-else class="avatar">{{ avatarText }}</span>
            <span class="user-name">{{ auth.user?.nickname || auth.user?.username }}</span>
            <el-icon class="user-arrow"><ArrowDown /></el-icon>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="profile">
                <el-icon><User /></el-icon>
                个人中心
              </el-dropdown-item>
              <el-dropdown-item command="logout" divided>
                <el-icon><SwitchButton /></el-icon>
                退出登录
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </el-header>
      <el-main class="main">
        <!-- 不包 transition:列表页模板是“卡片+弹窗”多根节点,与 out-in 过场组合会卡住挂载 -->
        <router-view />
      </el-main>
    </el-container>
    <ProfileDrawer v-model="profileOpen" />
  </el-container>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowDown, Expand, Fold, Platform, SwitchButton, User } from '@element-plus/icons-vue'
import { useAuthStore } from '../stores/auth'
import AdminMenuItem from './AdminMenuItem.vue'
import ProfileDrawer from './ProfileDrawer.vue'
import type { MenuNode } from '../api'

const auth = useAuthStore()
const router = useRouter()
const menus = ref<MenuNode[]>([])
const collapsed = ref(false)
const profileOpen = ref(false)

const avatarText = computed(() => {
  const name = auth.user?.nickname || auth.user?.username || '?'
  return name.slice(0, 1).toUpperCase()
})

onMounted(async () => {
  const me = await auth.fetchMe(true)
  menus.value = me?.menus ?? []
})

async function onCommand(cmd: string) {
  if (cmd === 'profile') {
    profileOpen.value = true
    return
  }
  if (cmd === 'logout') {
    await auth.logout()
    ElMessage.success('已退出登录')
    router.push('/login')
  }
}
</script>

<script lang="ts">
export default { name: 'AdminLayout' }
</script>

<style scoped>
.layout {
  height: 100vh;
}

.aside {
  display: flex;
  flex-direction: column;
  background: var(--app-aside-bg, #f9fafc);
  border-right: 1px solid var(--app-border, #e8e9ec);
  transition: width 0.25s ease;
  overflow: hidden;
}

.logo {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  height: 60px;
  flex-shrink: 0;
}

.logo-mark {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  border-radius: 9px;
  color: #fff;
  background: linear-gradient(135deg, #4f8bff, #2563eb);
  box-shadow: 0 4px 10px rgb(37 99 235 / 30%);
}

.logo-title {
  font-size: 16px;
  font-weight: 600;
  color: #1d2129;
  white-space: nowrap;
}

.menu {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 56px;
  padding: 0 16px;
  background: var(--app-page-bg, #f3f4f8);
  border-bottom: 1px solid var(--app-border, #e8e9ec);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 14px;
}

.collapse-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 7px;
  color: #4e5969;
  cursor: pointer;
  transition: background-color 0.2s, color 0.2s;
}

.collapse-btn:hover {
  background: #fff;
  color: var(--el-color-primary);
}

.user {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  outline: none;
}

.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  font-size: 13px;
  font-weight: 600;
  color: #fff;
  background: linear-gradient(135deg, #4f8bff, #2563eb);
}

.avatar--img {
  object-fit: cover;
  background: none;
}

.user-name {
  font-size: 14px;
  color: #4e5969;
}

.user-arrow {
  font-size: 12px;
  color: #86909c;
}

.main {
  padding: 10px;
  background: var(--app-page-bg, #f3f4f8);
  overflow-y: auto;
}
</style>
