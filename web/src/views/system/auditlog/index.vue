<template>
  <PaginatedTable ref="tableRef" :fetch="logApi.auditLogs" :query="filters">
    <template #toolbar>
      <el-input v-model="filters.username" placeholder="操作人" clearable style="width: 180px" @keyup.enter="tableRef?.load()" />
      <el-input v-model="filters.path" placeholder="接口路径" clearable style="width: 200px" @keyup.enter="tableRef?.load()" />
      <el-button type="primary" @click="tableRef?.load()">查询</el-button>
    </template>
    <el-table-column prop="id" label="ID" width="80" />
    <el-table-column prop="username" label="操作人" width="110" />
    <el-table-column prop="method" label="方法" width="90" />
    <el-table-column prop="path" label="接口" min-width="180" show-overflow-tooltip />
    <el-table-column label="状态码" width="90">
      <template #default="{ row }">
        <el-tag :type="row.status < 400 ? 'success' : 'danger'">{{ row.status }}</el-tag>
      </template>
    </el-table-column>
    <el-table-column prop="latencyMs" label="耗时(ms)" width="100" />
    <el-table-column prop="ip" label="IP" width="140" />
    <el-table-column prop="requestSummary" label="请求摘要" min-width="200" show-overflow-tooltip />
    <el-table-column prop="createdAt" label="时间" width="180" />
  </PaginatedTable>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { logApi } from '../../../api'
import PaginatedTable from '../../../components/PaginatedTable.vue'

const tableRef = ref<{ load: () => Promise<void> } | undefined>()
const filters = reactive({ username: '', path: '' })
</script>
