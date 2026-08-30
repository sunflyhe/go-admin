<template>
  <PageHeader description="记录账号登录成功与失败，含来源 IP 与浏览器。" />
  <PaginatedTable ref="tableRef" :fetch="logApi.loginLogs" :query="filters">
    <template #toolbar>
      <el-input v-model="filters.username" placeholder="用户名" clearable style="width: 180px" @keyup.enter="tableRef?.search()" />
      <el-select v-model="filters.success" placeholder="结果" clearable style="width: 120px">
        <el-option label="成功" :value="true" />
        <el-option label="失败" :value="false" />
      </el-select>
      <FilterActions @search="tableRef?.search()" @reset="resetFilters" />
    </template>
    <el-table-column label="#" type="index" :index="rowNumber" width="70" />
    <el-table-column prop="username" label="用户名" width="140" />
    <el-table-column prop="ip" label="登录IP" width="150" />
    <el-table-column prop="userAgent" label="浏览器" min-width="220" show-overflow-tooltip />
    <el-table-column label="登录状态" width="120">
      <template #default="{ row }">
        <!-- 失败原因不再单占列,悬停状态标签查看 -->
        <el-tooltip v-if="!row.success && row.failReason" :content="row.failReason" placement="top">
          <el-tag type="danger">失败</el-tag>
        </el-tooltip>
        <el-tag v-else :type="row.success ? 'success' : 'danger'">{{ row.success ? '成功' : '失败' }}</el-tag>
      </template>
    </el-table-column>
    <el-table-column label="登录时间" width="180">
      <template #default="{ row }">{{ formatDateTime(row.createdAt, true) }}</template>
    </el-table-column>
  </PaginatedTable>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { logApi } from '../../../api'
import { formatDateTime } from '../../../utils/format'
import PaginatedTable from '../../../components/PaginatedTable.vue'
import PageHeader from '../../../components/PageHeader.vue'
import FilterActions from '../../../components/FilterActions.vue'
import type { PaginatedTableHandle } from '../../../components/paginated-table'

const tableRef = ref<PaginatedTableHandle | undefined>()
const filters = reactive<{ username: string; success?: boolean }>({ username: '', success: undefined })

function resetFilters() {
  filters.username = ''
  filters.success = undefined
  tableRef.value?.search()
}

function rowNumber(index: number) {
  return ((tableRef.value?.page ?? 1) - 1) * (tableRef.value?.pageSize ?? 20) + index + 1
}
</script>
