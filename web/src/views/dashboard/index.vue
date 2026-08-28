<template>
  <el-card>
    <h3>欢迎使用 Go Admin 管理后台</h3>
    <p>当前用户:{{ auth.user?.nickname || auth.user?.username }}</p>
    <p>角色:{{ auth.user?.roles?.join(', ') || '-' }}</p>
  </el-card>
</template>

<script setup lang="ts">
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
</script>
