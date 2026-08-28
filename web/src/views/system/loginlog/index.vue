<template>
  <PaginatedTable ref="tableRef" :fetch="logApi.loginLogs" :query="filters">
    <template #toolbar>
      <el-input v-model="filters.username" placeholder="账号" clearable style="width: 180px" @keyup.enter="tableRef?.load()" />
      <el-select v-model="filters.success" placeholder="结果" clearable style="width: 120px">
        <el-option label="成功" :value="true" />
        <el-option label="失败" :value="false" />
      </el-select>
      <el-button type="primary" @click="tableRef?.load()">查询</el-button>
    </template>
    <el-table-column prop="id" label="ID" width="80" />
    <el-table-column prop="username" label="账号" />
    <el-table-column label="结果" width="90">
      <template #default="{ row }">
        <el-tag :type="row.success ? 'success' : 'danger'">{{ row.success ? '成功' : '失败' }}</el-tag>
      </template>
    </el-table-column>
    <el-table-column prop="failReason" label="失败原因" />
    <el-table-column prop="ip" label="IP" width="140" />
    <el-table-column prop="userAgent" label="User-Agent" min-width="180" show-overflow-tooltip />
    <el-table-column prop="createdAt" label="时间" width="180" />
  </PaginatedTable>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { logApi } from '../../../api'
import PaginatedTable from '../../../components/PaginatedTable.vue'

const tableRef = ref<{ load: () => Promise<void> } | undefined>()
const filters = reactive<{ username: string; success?: boolean }>({ username: '', success: undefined })
</script>
