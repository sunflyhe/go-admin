<template>
  <PaginatedTable ref="tableRef" :fetch="userApi.list" :query="filters">
    <template #toolbar>
      <el-input v-model="filters.username" placeholder="用户名" clearable style="width: 180px" @keyup.enter="tableRef?.load()" />
      <el-select v-model="filters.status" placeholder="状态" clearable style="width: 120px">
        <el-option label="启用" :value="1" />
        <el-option label="停用" :value="2" />
      </el-select>
      <el-button type="primary" @click="tableRef?.load()">查询</el-button>
      <el-button v-perm="'system:user:create'" type="success" @click="openCreate">新建用户</el-button>
      <el-button v-perm="'system:user:export'" @click="onExport">导出 Excel</el-button>
    </template>

    <el-table-column prop="id" label="ID" width="70" />
    <el-table-column prop="username" label="用户名" />
    <el-table-column prop="nickname" label="昵称" />
    <el-table-column prop="email" label="邮箱" />
    <el-table-column prop="phone" label="手机号" />
    <el-table-column label="状态" width="90">
      <template #default="{ row }">
        <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '启用' : '停用' }}</el-tag>
      </template>
    </el-table-column>
    <el-table-column prop="lastLoginAt" label="最后登录" width="170" />
    <el-table-column label="操作" width="330" fixed="right">
      <template #default="{ row }">
        <el-button v-perm="'system:user:update'" size="small" @click="openEdit(row)">编辑</el-button>
        <el-button v-perm="'system:user:reset-password'" size="small" @click="onResetPassword(row)">重置密码</el-button>
        <el-button v-perm="'system:user:assign-role'" size="small" @click="openRoles(row)">分配角色</el-button>
        <el-button
          v-perm="'system:user:update'"
          size="small"
          :type="row.status === 1 ? 'warning' : 'success'"
          :disabled="row.super"
          @click="onToggleStatus(row)"
        >
          {{ row.status === 1 ? '停用' : '启用' }}
        </el-button>
        <el-button v-perm="'system:user:delete'" size="small" type="danger" :disabled="row.super" @click="onDelete(row)">删除</el-button>
      </template>
    </el-table-column>
  </PaginatedTable>

  <!-- 新建/编辑 -->
  <el-dialog v-model="editVisible" :title="editForm.id ? '编辑用户' : '新建用户'" width="480px">
    <el-form :model="editForm" label-width="80px">
      <el-form-item label="用户名" required>
        <el-input v-model="editForm.username" :disabled="!!editForm.id" />
      </el-form-item>
      <el-form-item v-if="!editForm.id" label="密码" required>
        <el-input v-model="editForm.password" type="password" show-password placeholder="至少 8 位" />
      </el-form-item>
      <el-form-item label="昵称"><el-input v-model="editForm.nickname" /></el-form-item>
      <el-form-item label="邮箱"><el-input v-model="editForm.email" /></el-form-item>
      <el-form-item label="手机号"><el-input v-model="editForm.phone" /></el-form-item>
      <el-form-item label="状态">
        <el-switch v-model="editForm.status" :active-value="1" :inactive-value="2" active-text="启用" inactive-text="停用" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="editVisible = false">取消</el-button>
      <el-button type="primary" @click="save">保存</el-button>
    </template>
  </el-dialog>

  <!-- 分配角色 -->
  <el-dialog v-model="rolesVisible" title="分配角色" width="420px">
    <el-select v-model="selectedRoleIds" multiple style="width: 100%" placeholder="选择角色">
      <el-option v-for="r in allRoles" :key="r.id" :label="r.name" :value="r.id" />
    </el-select>
    <template #footer>
      <el-button @click="rolesVisible = false">取消</el-button>
      <el-button type="primary" @click="saveRoles">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { userApi, roleApi, type UserItem, type RoleItem } from '../../../api'
import PaginatedTable from '../../../components/PaginatedTable.vue'

const tableRef = ref<{ load: () => Promise<void> } | undefined>()
const filters = reactive<{ username: string; status?: number }>({ username: '', status: undefined })

const editVisible = ref(false)
const editForm = reactive({ id: 0, username: '', password: '', nickname: '', email: '', phone: '', status: 1 })

const rolesVisible = ref(false)
const selectedRoleIds = ref<number[]>([])
const allRoles = ref<RoleItem[]>([])
let rolesTarget = 0

function openCreate() {
  Object.assign(editForm, { id: 0, username: '', password: '', nickname: '', email: '', phone: '', status: 1 })
  editVisible.value = true
}

function openEdit(row: UserItem) {
  Object.assign(editForm, {
    id: row.id, username: row.username, password: '', nickname: row.nickname,
    email: row.email, phone: row.phone, status: row.status
  })
  editVisible.value = true
}

async function save() {
  if (editForm.id) {
    await userApi.update(editForm.id, {
      nickname: editForm.nickname, email: editForm.email, phone: editForm.phone, status: editForm.status
    })
  } else {
    await userApi.create({ ...editForm })
  }
  ElMessage.success('保存成功')
  editVisible.value = false
  tableRef.value?.load()
}

async function onDelete(row: UserItem) {
  await ElMessageBox.confirm(`确认删除用户「${row.username}」?`, '提示', { type: 'warning' })
  await userApi.remove(row.id)
  ElMessage.success('删除成功')
  tableRef.value?.load()
}

async function onToggleStatus(row: UserItem) {
  await userApi.setStatus(row.id, row.status === 1 ? 2 : 1)
  ElMessage.success('操作成功')
  tableRef.value?.load()
}

async function onResetPassword(row: UserItem) {
  const { value } = await ElMessageBox.prompt(`为「${row.username}」设置新密码(至少 8 位)`, '重置密码', { inputType: 'password' })
  if (!value || value.length < 8) {
    ElMessage.error('密码至少 8 位')
    return
  }
  await userApi.resetPassword(row.id, value)
  ElMessage.success('重置成功')
}

async function openRoles(row: UserItem) {
  rolesTarget = row.id
  selectedRoleIds.value = [...row.roleIds]
  const { data } = await roleApi.list({ page: 1, pageSize: 100 })
  allRoles.value = data.data.list.filter((r: RoleItem) => r.status === 1)
  rolesVisible.value = true
}

async function saveRoles() {
  await userApi.assignRoles(rolesTarget, selectedRoleIds.value)
  ElMessage.success('分配成功,该用户需重新登录生效')
  rolesVisible.value = false
  tableRef.value?.load()
}

function onExport() {
  const token = localStorage.getItem('accessToken')
  fetch(userApi.exportUrl, { headers: { Authorization: `Bearer ${token}` } })
    .then(async (res) => {
      if (!res.ok) throw new Error('导出失败')
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'users.xlsx'
      a.click()
      URL.revokeObjectURL(url)
    })
    .catch(() => ElMessage.error('导出失败'))
}
</script>
