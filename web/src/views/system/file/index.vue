<template>
  <el-card>
    <div class="toolbar">
      <el-input v-model="query.originName" placeholder="文件名" clearable style="width: 180px" @keyup.enter="load" />
      <el-upload
        :show-file-list="false"
        :auto-upload="false"
        :on-change="onFileChange"
        accept=".jpg,.jpeg,.png,.gif,.webp,.pdf,.txt,.csv,.xlsx,.xls,.docx,.zip"
      >
        <el-button v-perm="'system:file:upload'" type="success" :loading="uploading">上传文件</el-button>
      </el-upload>
    </div>
    <el-table :data="rows" border stripe v-loading="loading">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="originName" label="文件名" min-width="180" show-overflow-tooltip />
      <el-table-column prop="mime" label="类型" width="140" />
      <el-table-column label="大小" width="110">
        <template #default="{ row }">{{ formatSize(row.size) }}</template>
      </el-table-column>
      <el-table-column label="可见性" width="90">
        <template #default="{ row }">
          <el-tag :type="row.isPublic ? 'success' : 'info'">{{ row.isPublic ? '公开' : '私有' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="uploader" label="上传人" width="110" />
      <el-table-column prop="createdAt" label="时间" width="180" />
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" plain @click="onDownload(row)">下载</el-button>
          <el-button v-perm="'system:file:delete'" size="small" type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
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
import { ElMessage, ElMessageBox } from 'element-plus'
import type { UploadFile } from 'element-plus'
import { fileApi, type FileRow } from '../../../api'

const rows = ref<FileRow[]>([])
const total = ref(0)
const loading = ref(false)
const uploading = ref(false)
const query = reactive({ page: 1, pageSize: 20, originName: '' })

async function load() {
  loading.value = true
  try {
    const { data } = await fileApi.list({ ...query })
    rows.value = data.data.list
    total.value = data.data.total
  } finally {
    loading.value = false
  }
}

async function onFileChange(uploadFile: UploadFile) {
  if (!uploadFile.raw) return
  uploading.value = true
  try {
    const fd = new FormData()
    fd.append('file', uploadFile.raw)
    fd.append('isPublic', 'false')
    const token = localStorage.getItem('accessToken')
    const res = await fetch('/api/v1/files', { method: 'POST', headers: { Authorization: `Bearer ${token}` }, body: fd })
    const body = await res.json()
    if (body.code !== 0) {
      ElMessage.error(body.message || '上传失败')
    } else {
      ElMessage.success('上传成功')
      load()
    }
  } finally {
    uploading.value = false
  }
}

function onDownload(row: FileRow) {
  const url = row.isPublic ? `/files/${row.storePath}` : `/api/v1/files/${row.id}/download`
  const a = document.createElement('a')
  a.href = url
  a.download = row.originName
  a.click()
}

async function onDelete(row: FileRow) {
  await ElMessageBox.confirm(`确认删除「${row.originName}」?`, '提示', { type: 'warning' })
  await fileApi.remove(row.id)
  ElMessage.success('删除成功')
  load()
}

function formatSize(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(1)} MB`
}

onMounted(load)
</script>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; }
</style>
