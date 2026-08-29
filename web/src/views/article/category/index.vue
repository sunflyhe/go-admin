<template>
  <PageHeader description="维护文章分类，分类用于归纳文章，删除前需先清空其下文章。">
    <template #extra>
      <el-button v-perm="'article:category:create'" type="primary" @click="openCreate">新建分类</el-button>
    </template>
  </PageHeader>
  <el-card>
    <el-table v-loading="loading" :data="categories">
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="name" label="分类名" min-width="180" />
      <el-table-column prop="sort" label="排序" width="90" align="right" />
      <el-table-column prop="count" label="文章数" width="90" align="right" />
      <el-table-column label="创建时间" width="150">
        <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="140" fixed="right" align="center">
        <template #default="{ row }">
          <el-button v-perm="'article:category:update'" link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button v-perm="'article:category:delete'" link type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog v-model="editVisible" :title="editForm.id ? '编辑分类' : '新建分类'" width="420px">
    <el-form :model="editForm" label-width="70px">
      <el-form-item label="分类名" required><el-input v-model="editForm.name" maxlength="64" /></el-form-item>
      <el-form-item label="排序">
        <el-input-number v-model="editForm.sort" :min="0" controls-position="right" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="editVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="save">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { articleCategoryApi, type ArticleCategory } from '../../../api'
import PageHeader from '../../../components/PageHeader.vue'
import { formatDateTime } from '../../../utils/format'

// 分类是少量枚举,不分页;文章编辑弹窗的下拉也复用 list 接口。
const categories = ref<ArticleCategory[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const { data } = await articleCategoryApi.list()
    categories.value = data.data
  } finally {
    loading.value = false
  }
}
onMounted(load)

const editVisible = ref(false)
const saving = ref(false)
const editForm = reactive({ id: 0, name: '', sort: 0 })

function openCreate() {
  Object.assign(editForm, { id: 0, name: '', sort: 0 })
  editVisible.value = true
}

function openEdit(row: ArticleCategory) {
  Object.assign(editForm, { id: row.id, name: row.name, sort: row.sort })
  editVisible.value = true
}

async function save() {
  if (!editForm.name.trim()) {
    ElMessage.warning('请输入分类名')
    return
  }
  saving.value = true
  try {
    const payload = { name: editForm.name.trim(), sort: editForm.sort }
    if (editForm.id) {
      await articleCategoryApi.update(editForm.id, payload)
    } else {
      await articleCategoryApi.create(payload)
    }
    ElMessage.success('保存成功')
    editVisible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

async function onDelete(row: ArticleCategory) {
  await ElMessageBox.confirm(`确认删除分类「${row.name}」?`, '提示', { type: 'warning' })
  await articleCategoryApi.remove(row.id)
  ElMessage.success('删除成功')
  await load()
}
</script>
