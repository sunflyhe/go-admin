<template>
  <PageHeader description="维护系统参数键值对，业务模块按参数键读取取值；内置参数可改值，不可改键名或删除。">
    <template #extra>
      <el-button v-perm="'system:config:create'" type="primary" @click="openCreate">新建参数</el-button>
    </template>
  </PageHeader>
  <PaginatedTable ref="tableRef" :fetch="configApi.list" :query="query">
    <template #toolbar>
      <el-input
        v-model="query.keyword"
        placeholder="参数名 / 参数键"
        clearable
        style="width: 220px"
        @keyup.enter="tableRef?.search()"
      />
      <FilterActions @search="tableRef?.search()" @reset="onResetFilters" />
    </template>
    <el-table-column prop="id" label="ID" width="70" />
    <el-table-column prop="name" label="参数名" min-width="140" />
    <el-table-column prop="key" label="参数键" min-width="160" />
    <el-table-column prop="value" label="参数值" min-width="150" show-overflow-tooltip />
    <el-table-column prop="remark" label="备注" min-width="170" show-overflow-tooltip />
    <el-table-column label="内置" width="80" align="center">
      <template #default="{ row }">
        <el-tag v-if="row.builtin" type="warning" size="small">内置</el-tag>
        <span v-else>-</span>
      </template>
    </el-table-column>
    <el-table-column label="更新时间" width="150">
      <template #default="{ row }">{{ formatDateTime(row.updatedAt) }}</template>
    </el-table-column>
    <el-table-column label="操作" width="140" fixed="right" align="center">
      <template #default="{ row }">
        <el-button v-perm="'system:config:update'" link type="primary" @click="openEdit(row)">编辑</el-button>
        <el-button
          v-perm="'system:config:delete'"
          link
          type="danger"
          :disabled="row.builtin"
          @click="onDelete(row)"
        >删除</el-button>
      </template>
    </el-table-column>
  </PaginatedTable>

  <el-dialog v-model="editVisible" :title="editForm.id ? '编辑参数' : '新建参数'" width="480px">
    <el-form :model="editForm" label-width="80px">
      <el-form-item label="参数名" required><el-input v-model="editForm.name" maxlength="64" /></el-form-item>
      <el-form-item label="参数键" required>
        <el-input v-model="editForm.key" maxlength="64" :disabled="isBuiltinEdit" placeholder="小写字母开头,可用 . - _" />
      </el-form-item>
      <el-form-item label="参数值">
        <el-input v-model="editForm.value" type="textarea" :rows="3" maxlength="512" show-word-limit />
      </el-form-item>
      <el-form-item label="备注">
        <el-input v-model="editForm.remark" type="textarea" :rows="2" maxlength="255" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="editVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="save">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { configApi, type SysConfig } from '../../../api'
import FilterActions from '../../../components/FilterActions.vue'
import PaginatedTable from '../../../components/PaginatedTable.vue'
import PageHeader from '../../../components/PageHeader.vue'
import { formatDateTime } from '../../../utils/format'
import type { PaginatedTableHandle } from '../../../components/paginated-table'

// 关键字同时匹配参数名与参数键,由服务端模糊查询;search 会回到第一页,避免停在旧页码。
const tableRef = ref<PaginatedTableHandle | undefined>()
const query = reactive<Record<string, unknown>>({ keyword: '' })

function onResetFilters() {
  query.keyword = ''
  tableRef.value?.search()
}

const editVisible = ref(false)
const saving = ref(false)
const editForm = reactive({ id: 0, name: '', key: '', value: '', remark: '', builtin: false })
// 内置参数只允许改名称、值与备注,键名由服务端保护,前端直接禁用输入框
const isBuiltinEdit = computed(() => editForm.id > 0 && editForm.builtin)

function openCreate() {
  Object.assign(editForm, { id: 0, name: '', key: '', value: '', remark: '', builtin: false })
  editVisible.value = true
}

function openEdit(row: SysConfig) {
  Object.assign(editForm, {
    id: row.id, name: row.name, key: row.key, value: row.value, remark: row.remark, builtin: row.builtin
  })
  editVisible.value = true
}

async function save() {
  if (!editForm.name.trim()) {
    ElMessage.warning('请输入参数名')
    return
  }
  if (!editForm.key.trim()) {
    ElMessage.warning('请输入参数键')
    return
  }
  saving.value = true
  try {
    const payload = {
      name: editForm.name.trim(),
      key: editForm.key.trim(),
      value: editForm.value,
      remark: editForm.remark.trim()
    }
    if (editForm.id) {
      await configApi.update(editForm.id, payload)
    } else {
      await configApi.create(payload)
    }
    ElMessage.success('保存成功')
    editVisible.value = false
    await tableRef.value?.load()
  } finally {
    saving.value = false
  }
}

async function onDelete(row: SysConfig) {
  await ElMessageBox.confirm(`确认删除参数「${row.name}」?`, '提示', { type: 'warning' })
  await configApi.remove(row.id)
  ElMessage.success('删除成功')
  await tableRef.value?.load()
}
</script>
