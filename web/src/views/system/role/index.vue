<template>
  <PageHeader description="维护角色及其菜单与按钮权限，授权即时生效。">
    <template #extra>
      <el-button v-perm="'system:role:create'" type="primary" @click="openCreate">新建角色</el-button>
    </template>
  </PageHeader>
  <PaginatedTable ref="tableRef" :fetch="roleApi.list" :query="filters">
    <template #toolbar>
      <el-input v-model="filters.name" placeholder="角色名" clearable style="width: 180px" @keyup.enter="tableRef?.search()" />
      <FilterActions @search="tableRef?.search()" @reset="resetFilters" />
    </template>

    <el-table-column prop="id" label="ID" width="70" />
    <el-table-column label="角色名" min-width="150">
      <template #default="{ row }">
        {{ row.name }}
        <el-tag v-if="row.builtin" type="info" size="small" class="builtin-tag">内置</el-tag>
      </template>
    </el-table-column>
    <el-table-column prop="code" label="编码" width="130" />
    <el-table-column prop="description" label="描述" min-width="160" show-overflow-tooltip />
    <el-table-column prop="userCount" label="成员数" width="90" align="right" />
    <el-table-column label="状态" width="90" align="center">
      <template #default="{ row }">
        <el-switch
          :model-value="row.status === 1"
          :disabled="row.builtin || !auth.hasPerm('system:role:update')"
          :before-change="() => confirmToggleStatus(row)"
          @change="() => toggleStatus(row)"
        />
      </template>
    </el-table-column>
    <el-table-column label="操作" width="190" fixed="right" align="center">
      <template #default="{ row }">
        <el-button v-perm="'system:role:update'" link type="primary" @click="openEdit(row)">编辑</el-button>
        <el-button v-perm="'system:role:assign-menu'" link type="primary" @click="openMenus(row)">分配菜单</el-button>
        <el-button v-perm="'system:role:delete'" link type="danger" :disabled="row.builtin" @click="onDelete(row)">删除</el-button>
      </template>
    </el-table-column>
  </PaginatedTable>

  <el-dialog v-model="editVisible" :title="editForm.id ? '编辑角色' : '新建角色'" width="480px">
    <el-form :model="editForm" label-width="80px">
      <el-form-item label="角色名" required><el-input v-model="editForm.name" /></el-form-item>
      <el-form-item label="编码" required>
        <el-input v-model="editForm.code" :disabled="!!editForm.id" placeholder="如 ops,创建后不可修改" />
      </el-form-item>
      <el-form-item label="描述"><el-input v-model="editForm.description" type="textarea" /></el-form-item>
      <el-form-item label="状态">
        <el-switch v-model="editForm.status" :active-value="1" :inactive-value="2" active-text="启用" inactive-text="停用" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="editVisible = false">取消</el-button>
      <el-button type="primary" @click="save">保存</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="menusVisible" title="分配菜单" width="480px">
    <div class="menu-tree-card">
      <el-tree
        ref="menuTreeRef"
        :data="menuTree"
        show-checkbox
        node-key="id"
        :props="{ label: 'name', children: 'children' }"
        default-expand-all
      />
    </div>
    <template #footer>
      <el-button @click="menusVisible = false">取消</el-button>
      <el-button type="primary" @click="saveMenus">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { nextTick, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { roleApi, menuApi, type RoleItem, type MenuNode } from '../../../api'
import { useAuthStore } from '../../../stores/auth'
import PaginatedTable from '../../../components/PaginatedTable.vue'
import PageHeader from '../../../components/PageHeader.vue'
import FilterActions from '../../../components/FilterActions.vue'
import type { PaginatedTableHandle } from '../../../components/paginated-table'

const tableRef = ref<PaginatedTableHandle | undefined>()
const auth = useAuthStore()
const filters = reactive({ name: '' })

function resetFilters() {
  filters.name = ''
  tableRef.value?.search()
}

// before-change 拦截停用方向:停用角色会影响其下所有账号的权限,需二次确认;启用直接放行。
// 内置超管角色的开关已按 builtin 禁用,后端同样拒绝停用。
function confirmToggleStatus(row: RoleItem) {
  if (row.status !== 1) return Promise.resolve(true)
  return ElMessageBox.confirm(`确认停用角色「${row.name}」?停用后该角色下账号将失去对应权限。`, '提示', { type: 'warning' })
}

async function toggleStatus(row: RoleItem) {
  await roleApi.update(row.id, { name: row.name, code: row.code, description: row.description, status: row.status === 1 ? 2 : 1 })
  ElMessage.success('操作成功')
  tableRef.value?.load()
}

const editVisible = ref(false)
const editForm = reactive({ id: 0, name: '', code: '', description: '', status: 1 })

const menusVisible = ref(false)
const menuTree = ref<MenuNode[]>([])
const menuTreeRef = ref<{ getCheckedKeys: (b: boolean) => number[]; setCheckedKeys: (k: number[]) => void }>()
let menusTarget = 0

function openCreate() {
  Object.assign(editForm, { id: 0, name: '', code: '', description: '', status: 1 })
  editVisible.value = true
}

function openEdit(row: RoleItem) {
  Object.assign(editForm, { id: row.id, name: row.name, code: row.code, description: row.description, status: row.status })
  editVisible.value = true
}

async function save() {
  if (editForm.id) {
    await roleApi.update(editForm.id, { ...editForm })
  } else {
    await roleApi.create({ ...editForm })
  }
  ElMessage.success('保存成功')
  editVisible.value = false
  tableRef.value?.load()
}

async function onDelete(row: RoleItem) {
  await ElMessageBox.confirm(`确认删除角色「${row.name}」?`, '提示', { type: 'warning' })
  await roleApi.remove(row.id)
  ElMessage.success('删除成功')
  tableRef.value?.load()
}

async function openMenus(row: RoleItem) {
  menusTarget = row.id
  const [treeRes, ownRes] = await Promise.all([menuApi.tree(), roleApi.menus(row.id)])
  menuTree.value = treeRes.data.data
  await nextTick()
  menuTreeRef.value?.setCheckedKeys(ownRes.data.data)
  menusVisible.value = true
}

async function saveMenus() {
  const checked = menuTreeRef.value?.getCheckedKeys(true) ?? []
  await roleApi.assignMenus(menusTarget, checked)
  ElMessage.success('分配成功')
  menusVisible.value = false
  tableRef.value?.load()
}
</script>

<style scoped>
.builtin-tag {
  margin-left: 6px;
}

.menu-tree-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 12px;
  background: #f8fafc;
  max-height: 360px;
  overflow-y: auto;
}

.menu-tree-card :deep(.el-tree) {
  background: transparent;
}
</style>
