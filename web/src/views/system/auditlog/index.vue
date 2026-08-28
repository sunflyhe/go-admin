<template>
  <el-card>
    <div class="toolbar">
      <el-input v-model="query.username" placeholder="操作人" clearable style="width: 180px" @keyup.enter="load" />
      <el-input v-model="query.path" placeholder="接口路径" clearable style="width: 200px" @keyup.enter="load" />
      <el-button type="primary" @click="load">查询</el-button>
    </div>
    <el-table :data="rows" border stripe v-loading="loading">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="username" label="操作人" width="110" />
      <el-table-column prop="method" label="方法" width="90" />
      <el-table-column prop="path" label="接口" min-width="180" show-overflow-tooltip />
      <el-table-column prop="status" label="状态码" width="90">
        <template #default="{ row }">
          <el-tag :type="row.status < 400 ? 'success' : 'danger'">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="latencyMs" label="耗时(ms)" width="100" />
      <el-table-column prop="ip" label="IP" width="140" />
      <el-table-column prop="requestSummary" label="请求摘要" min-width="200" show-overflow-tooltip />
      <el-table-column prop="createdAt" label="时间" width="180" />
    </el-table>
    <el-pagination
      v-model:current-page="query.page"
      v-model:page-size="query.pageSize"
      :total="total"
      layout="total, prev, pager, next"
      style="margin-top: 12px; justify-content: flex-end"
      @current-change="load"
    />
  </el-card>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { logApi, type AuditLog } from '../../../api'

const rows = ref<AuditLog[]>([])
const total = ref(0)
const loading = ref(false)
const query = reactive({ page: 1, pageSize: 20, username: '', path: '' })

async function load() {
  loading.value = true
  try {
    const { data } = await logApi.auditLogs({ ...query })
    rows.value = data.data.list
    total.value = data.data.total
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; }
</style>
