<template>
  <el-card>
    <div class="toolbar">
      <el-input v-model="query.name" placeholder="角色名" clearable style="width: 180px" @keyup.enter="load" />
      <el-button type="primary" @click="load">查询</el-button>
      <el-button v-perm="'system:role:create'" type="success" @click="openCreate">新建角色</el-button>
    </div>

    <el-table :data="rows" border stripe v-loading="loading">
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="name" label="角色名" />
      <el-table-column prop="code" label="编码" />
      <el-table-column prop="description" label="描述" />
      <el-table-column prop="userCount" label="用户数" width="90" />
      <el-table-column label="内置" width="80">
        <template #default="{ row }">
          <el-tag v-if="row.builtin" type="info">内置</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '启用' : '停用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="260" fixed="right">
        <template #default="{ row }">
          <el-button v-perm="'system:role:update'" size="small" @click="openEdit(row)">编辑</el-button>
          <el-button v-perm="'system:role:assign-menu'" size="small" type="primary" plain @click="openMenus(row)">分配菜单</el-button>
          <el-button v-perm="'system:role:delete'" size="small" type="danger" :disabled="row.builtin" @click="onDelete(row)">删除</el-button>
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
    <el-tree
      ref="menuTreeRef"
      :data="menuTree"
      show-checkbox
      node-key="id"
      :props="{ label: 'name', children: 'children' }"
      default-expand-all
    />
    <template #footer>
      <el-button @click="menusVisible = false">取消</el-button>
      <el-button type="primary" @click="saveMenus">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { nextTick, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { roleApi, menuApi, type RoleItem, type MenuNode } from '../../../api'

const rows = ref<RoleItem[]>([])
const total = ref(0)
const loading = ref(false)
const query = reactive({ page: 1, pageSize: 20, name: '' })

const editVisible = ref(false)
const editForm = reactive({ id: 0, name: '', code: '', description: '', status: 1 })

const menusVisible = ref(false)
const menuTree = ref<MenuNode[]>([])
const menuTreeRef = ref<{ getCheckedKeys: (b: boolean) => number[]; setCheckedKeys: (k: number[]) => void }>()
let menusTarget = 0

async function load() {
  loading.value = true
  try {
    const { data } = await roleApi.list({ ...query })
    rows.value = data.data.list
    total.value = data.data.total
  } finally {
    loading.value = false
  }
}

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
  load()
}

async function onDelete(row: RoleItem) {
  await ElMessageBox.confirm(`确认删除角色「${row.name}」?`, '提示', { type: 'warning' })
  await roleApi.remove(row.id)
  ElMessage.success('删除成功')
  load()
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
  load()
}

onMounted(load)
</script>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; }
</style>
