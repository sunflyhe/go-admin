<template>
  <el-card>
    <div v-if="$slots.toolbar" class="toolbar">
      <slot name="toolbar" />
    </div>
    <el-table v-loading="loading" :data="rows" border stripe>
      <slot />
    </el-table>
    <el-pagination
      v-model:current-page="page"
      v-model:page-size="pageSize"
      :total="total"
      layout="total, prev, pager, next"
      class="pager"
      @current-change="load"
    />
  </el-card>
</template>

<script setup lang="ts" generic="T">
import { onMounted, ref } from 'vue'
import type { ApiBody, Paged } from '../api'

// 通用分页列表:统一 loading、表格与分页交互;列定义由使用方以 el-table-column 子组件传入,
// 工具栏(筛选/操作按钮)通过 #toolbar 插槽注入;刷新通过模板 ref 调用 load()。
const props = defineProps<{
  fetch: (params: Record<string, unknown>) => Promise<{ data: ApiBody<Paged<T>> }>
  query?: Record<string, unknown>
}>()

const rows = ref<T[]>([]) as { value: T[] }
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const { data } = await props.fetch({ page: page.value, pageSize: pageSize.value, ...props.query })
    rows.value = data.data.list
    total.value = data.data.total
  } finally {
    loading.value = false
  }
}

defineExpose({ load })

onMounted(load)
</script>

<style scoped>
.toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}
.pager {
  margin-top: 12px;
  justify-content: flex-end;
}
</style>
