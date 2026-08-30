<template>
  <PageHeader description="记录写操作的操作人、接口、耗时与脱敏后的请求摘要。" />
  <PaginatedTable ref="tableRef" :fetch="logApi.auditLogs" :query="filters">
    <template #toolbar>
      <el-input v-model="filters.username" placeholder="操作人" clearable style="width: 180px" @keyup.enter="tableRef?.search()" />
      <el-input v-model="filters.path" placeholder="接口路径" clearable style="width: 200px" @keyup.enter="tableRef?.search()" />
      <FilterActions @search="tableRef?.search()" @reset="resetFilters" />
    </template>
    <el-table-column prop="id" label="ID" width="80" />
    <el-table-column prop="username" label="操作人" width="110" />
    <el-table-column label="接口" min-width="240" show-overflow-tooltip>
      <template #default="{ row }">
        <el-tag :type="methodTag(row.method)" size="small" class="method-tag">{{ row.method }}</el-tag>
        <span>{{ row.path }}</span>
      </template>
    </el-table-column>
    <el-table-column label="状态码" width="90">
      <template #default="{ row }">
        <el-tag :type="row.status < 400 ? 'success' : 'danger'">{{ row.status }}</el-tag>
      </template>
    </el-table-column>
    <el-table-column label="耗时" width="90" align="right">
      <template #default="{ row }">
        <span :class="{ slow: row.latencyMs > 1000 }">{{ row.latencyMs }}</span>
      </template>
    </el-table-column>
    <el-table-column prop="ip" label="IP" width="140" />
    <el-table-column prop="requestSummary" label="请求摘要" min-width="200" show-overflow-tooltip />
    <el-table-column label="时间" width="150">
      <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
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
const filters = reactive({ username: '', path: '' })

function resetFilters() {
  filters.username = ''
  filters.path = ''
  tableRef.value?.search()
}

// HTTP 方法 tag 配色:读=灰 写=蓝 改=橙 删=红
function methodTag(method: string): string {
  const map: Record<string, string> = { GET: 'info', POST: 'primary', PUT: 'warning', PATCH: 'warning', DELETE: 'danger' }
  return map[method] ?? 'info'
}
</script>

<style scoped>
.method-tag {
  margin-right: 8px;
}

.slow {
  color: var(--el-color-danger);
  font-weight: 500;
}
</style>
