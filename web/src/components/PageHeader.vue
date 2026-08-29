<template>
  <el-card class="page-header">
    <div class="page-header-inner">
      <div class="page-header-text">
        <h2 class="page-title">{{ displayTitle }}</h2>
        <p v-if="description" class="page-desc">{{ description }}</p>
      </div>
      <div v-if="$slots.extra" class="page-header-extra">
        <slot name="extra" />
      </div>
    </div>
  </el-card>
</template>

<script setup lang="ts">
// 菜单页标题栏:标题默认取路由 meta.title(即后端下发的菜单名),页面只需补描述与 #extra 操作按钮。
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const props = defineProps<{ title?: string; description?: string }>()
const route = useRoute()

const displayTitle = computed(() => props.title || (route.meta.title as string | undefined) || '')
</script>

<style scoped>
.page-header {
  margin-bottom: 10px;
}

.page-header-inner {
  display: flex;
  align-items: center;
  gap: 12px;
}

.page-header-text {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 12px;
}

.page-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.page-desc {
  margin: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.page-header-extra {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-left: auto;
}
</style>
