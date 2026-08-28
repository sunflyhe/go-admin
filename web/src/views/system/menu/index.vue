<template>
  <el-card>
    <div class="toolbar">
      <el-button v-perm="'system:menu:create'" type="success" @click="openCreate(0)">新建根菜单</el-button>
    </div>
    <el-table :data="tree" border row-key="id" default-expand-all v-loading="loading">
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column label="类型" width="90">
        <template #default="{ row }">
          <el-tag :type="['', 'info', 'success', 'warning'][row.type]">{{ ['', '目录', '页面', '按钮'][row.type] }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="path" label="路由" min-width="120" />
      <el-table-column prop="component" label="组件" min-width="140" />
      <el-table-column prop="permission" label="权限码" min-width="160" />
      <el-table-column prop="sort" label="排序" width="70" />
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '启用' : '停用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <el-button v-if="row.type !== 3" v-perm="'system:menu:create'" size="small" @click="openCreate(row.id)">添加子级</el-button>
          <el-button v-perm="'system:menu:update'" size="small" @click="openEdit(row)">编辑</el-button>
          <el-button v-perm="'system:menu:delete'" size="small" type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog v-model="editVisible" :title="editForm.id ? '编辑菜单' : '新建菜单'" width="520px">
    <el-form :model="editForm" label-width="90px">
      <el-form-item label="上级菜单">
        <el-tree-select
          v-model="editForm.parentId"
          :data="parentOptions"
          check-strictly
          :props="{ label: 'name', value: 'id' }"
          style="width: 100%"
          placeholder="无(根节点)"
          clearable
        />
      </el-form-item>
      <el-form-item label="类型" required>
        <el-radio-group v-model="editForm.type">
          <el-radio-button :value="1">目录</el-radio-button>
          <el-radio-button :value="2">页面</el-radio-button>
          <el-radio-button :value="3">按钮</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="名称" required><el-input v-model="editForm.name" /></el-form-item>
      <el-form-item v-if="editForm.type !== 3" label="路由"><el-input v-model="editForm.path" /></el-form-item>
      <el-form-item v-if="editForm.type === 2" label="组件"><el-input v-model="editForm.component" placeholder="如 system/user/index" /></el-form-item>
      <el-form-item label="权限码"><el-input v-model="editForm.permission" placeholder="如 system:user:create" /></el-form-item>
      <el-form-item label="图标"><el-input v-model="editForm.icon" /></el-form-item>
      <el-form-item label="排序"><el-input-number v-model="editForm.sort" :min="0" /></el-form-item>
      <el-form-item label="状态">
        <el-switch v-model="editForm.status" :active-value="1" :inactive-value="2" active-text="启用" inactive-text="停用" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="editVisible = false">取消</el-button>
      <el-button type="primary" @click="save">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { menuApi, type MenuRow, type MenuNode } from '../../../api'

const tree = ref<MenuNode[]>([])
const flat = ref<MenuRow[]>([])
const loading = ref(false)

const editVisible = ref(false)
const editForm = reactive({
  id: 0, parentId: 0, type: 2, name: '', path: '', component: '',
  permission: '', icon: '', sort: 0, status: 1
})

const parentOptions = computed(() => tree.value)

async function load() {
  loading.value = true
  try {
    const [treeRes, listRes] = await Promise.all([menuApi.tree(), menuApi.list()])
    tree.value = treeRes.data.data
    flat.value = listRes.data.data
  } finally {
    loading.value = false
  }
}

function openCreate(parentId: number) {
  Object.assign(editForm, {
    id: 0, parentId, type: parentId === 0 ? 1 : 3, name: '', path: '', component: '',
    permission: '', icon: '', sort: 0, status: 1
  })
  editVisible.value = true
}

function openEdit(row: MenuRow) {
  Object.assign(editForm, {
    id: row.id, parentId: row.parentId, type: row.type, name: row.name, path: row.path,
    component: row.component, permission: row.permission, icon: row.icon, sort: row.sort, status: row.status
  })
  editVisible.value = true
}

async function save() {
  const payload = { ...editForm, parentId: editForm.parentId || 0 }
  if (editForm.id) {
    await menuApi.update(editForm.id, payload)
  } else {
    await menuApi.create(payload)
  }
  ElMessage.success('保存成功')
  editVisible.value = false
  load()
}

async function onDelete(row: MenuNode) {
  await ElMessageBox.confirm(`确认删除「${row.name}」?`, '提示', { type: 'warning' })
  await menuApi.remove(row.id)
  ElMessage.success('删除成功')
  load()
}

onMounted(load)
</script>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; }
</style>
