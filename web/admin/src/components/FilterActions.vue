<template>
  <div class="filter-actions">
    <el-button type="primary" plain :icon="Search" @click="emit('search')">搜索</el-button>
    <el-button plain :icon="RefreshLeft" @click="emit('reset')">重置</el-button>
  </div>
</template>

<script setup lang="ts">
// 列表页检索按钮组:统一轻风格(plain)并整体靠右;
// 重置只抛事件,清空哪些筛选条件由使用方自己决定。
import { RefreshLeft, Search } from '@element-plus/icons-vue'

const emit = defineEmits<{ search: []; reset: [] }>()
</script>

<style scoped>
.filter-actions {
  display: flex;
  align-items: center;
  margin-left: auto;
}
</style>
