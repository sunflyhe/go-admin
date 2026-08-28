<template>
  <el-container class="layout">
    <el-aside width="220px" class="aside">
      <div class="logo">Go Admin</div>
      <el-menu
        :default-active="$route.path"
        router
        background-color="#001529"
        text-color="#c8c9cc"
        active-text-color="#ffffff"
        class="menu"
      >
        <template v-for="node in menus" :key="node.id">
          <!-- 目录 -->
          <el-sub-menu v-if="node.type === 1 && node.children?.length" :index="node.path">
            <template #title>
              <el-icon v-if="node.icon"><component :is="node.icon" /></el-icon>
              <span>{{ node.name }}</span>
            </template>
            <el-menu-item v-for="child in node.children" :key="child.id" :index="child.path">
              <el-icon v-if="child.icon"><component :is="child.icon" /></el-icon>
              <span>{{ child.name }}</span>
            </el-menu-item>
          </el-sub-menu>
          <!-- 页面 -->
          <el-menu-item v-else-if="node.type === 2" :index="node.path">
            <el-icon v-if="node.icon"><component :is="node.icon" /></el-icon>
            <span>{{ node.name }}</span>
          </el-menu-item>
        </template>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <el-breadcrumb separator="/">
          <el-breadcrumb-item :to="{ path: '/dashboard' }">首页</el-breadcrumb-item>
          <el-breadcrumb-item v-if="$route.meta.title">{{ $route.meta.title }}</el-breadcrumb-item>
        </el-breadcrumb>
        <el-dropdown @command="onCommand">
          <span class="user">
            <el-icon><UserFilled /></el-icon>
            {{ auth.user?.nickname || auth.user?.username }}
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="logout">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </el-header>
      <el-main class="main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { UserFilled } from '@element-plus/icons-vue'
import { useAuthStore } from '../stores/auth'
import type { MenuNode } from '../api'

const auth = useAuthStore()
const router = useRouter()
const menus = ref<MenuNode[]>([])

onMounted(async () => {
  const me = await auth.fetchMe(true)
  menus.value = me?.menus ?? []
})

async function onCommand(cmd: string) {
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
.layout { height: 100vh; }
.aside { background-color: #001529; }
.logo { color: #fff; font-size: 18px; font-weight: 600; text-align: center; line-height: 56px; }
.menu { border-right: none; }
.header { display: flex; align-items: center; justify-content: space-between; border-bottom: 1px solid #eee; background: #fff; }
.user { cursor: pointer; display: inline-flex; align-items: center; gap: 4px; }
.main { background: #f5f7fa; }
</style>
