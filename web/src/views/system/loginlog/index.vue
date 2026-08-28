<template>
  <el-card>
    <div class="toolbar">
      <el-input v-model="query.username" placeholder="账号" clearable style="width: 180px" @keyup.enter="load" />
      <el-select v-model="query.success" placeholder="结果" clearable style="width: 120px">
        <el-option label="成功" :value="true" />
        <el-option label="失败" :value="false" />
      </el-select>
      <el-button type="primary" @click="load">查询</el-button>
    </div>
    <el-table :data="rows" border stripe v-loading="loading">
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
import { logApi, type LoginLog } from '../../../api'

const rows = ref<LoginLog[]>([])
const total = ref(0)
const loading = ref(false)
const query = reactive<{ page: number; pageSize: number; username: string; success?: boolean }>({
  page: 1, pageSize: 20, username: '', success: undefined
})

async function load() {
  loading.value = true
  try {
    const { data } = await logApi.loginLogs({ ...query })
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
