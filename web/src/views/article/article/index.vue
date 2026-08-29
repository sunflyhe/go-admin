<template>
  <PageHeader description="管理文章资讯：支持富文本正文与配图，草稿仅后台可见，发布后记录首发时间。">
    <template #extra>
      <el-button v-perm="'article:article:create'" type="primary" @click="openCreate">新建文章</el-button>
    </template>
  </PageHeader>
  <PaginatedTable ref="tableRef" :fetch="articleApi.list" :query="filters">
    <template #toolbar>
      <el-input v-model="filters.title" placeholder="标题" clearable style="width: 180px" @keyup.enter="tableRef?.search()" />
      <el-select v-model="filters.categoryId" placeholder="全部分类" clearable style="width: 150px">
        <el-option label="未分类" :value="0" />
        <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
      </el-select>
      <el-select v-model="filters.status" placeholder="全部状态" clearable style="width: 120px">
        <el-option label="草稿" :value="1" />
        <el-option label="已发布" :value="2" />
      </el-select>
      <FilterActions @search="tableRef?.search()" @reset="resetFilters" />
    </template>

    <el-table-column prop="id" label="ID" width="70" />
    <el-table-column prop="title" label="标题" min-width="200" show-overflow-tooltip />
    <el-table-column prop="categoryName" label="分类" width="120" show-overflow-tooltip />
    <el-table-column label="状态" width="90">
      <template #default="{ row }">
        <el-tag :type="row.status === 2 ? 'success' : 'info'">{{ row.status === 2 ? '已发布' : '草稿' }}</el-tag>
      </template>
    </el-table-column>
    <el-table-column prop="author" label="作者" width="110" />
    <el-table-column label="发布时间" width="150">
      <template #default="{ row }">{{ formatDateTime(row.publishedAt) }}</template>
    </el-table-column>
    <el-table-column label="创建时间" width="150">
      <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
    </el-table-column>
    <el-table-column label="操作" width="140" fixed="right" align="center">
      <template #default="{ row }">
        <el-button v-perm="'article:article:update'" link type="primary" @click="openEdit(row)">编辑</el-button>
        <el-button v-perm="'article:article:delete'" link type="danger" @click="onDelete(row)">删除</el-button>
      </template>
    </el-table-column>
  </PaginatedTable>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { articleApi, articleCategoryApi, type ArticleCategory, type ArticleRow } from '../../../api'
import PaginatedTable from '../../../components/PaginatedTable.vue'
import PageHeader from '../../../components/PageHeader.vue'
import FilterActions from '../../../components/FilterActions.vue'
import { formatDateTime } from '../../../utils/format'
import type { PaginatedTableHandle } from '../../../components/paginated-table'

const router = useRouter()
const tableRef = ref<PaginatedTableHandle | undefined>()
// categoryId:不选=全部;0(未分类)与分类 id 都是合法筛选值
const filters = reactive<{ title: string; categoryId: number | undefined; status: number | undefined }>({
  title: '',
  categoryId: undefined,
  status: undefined
})

const categories = ref<ArticleCategory[]>([])

// 分类下拉只服务筛选栏;无 article:category:list 权限时加载失败静默降级为"未分类"
onMounted(async () => {
  try {
    const { data } = await articleCategoryApi.list()
    categories.value = data.data
  } catch {
    categories.value = []
  }
})

function resetFilters() {
  Object.assign(filters, { title: '', categoryId: undefined, status: undefined })
  tableRef.value?.search()
}

// 新建/编辑走独立页面 /article/article/edit?id=xx;返回列表时重新挂载即刷新数据
function openCreate() {
  router.push('/article/article/edit')
}

function openEdit(row: ArticleRow) {
  router.push({ path: '/article/article/edit', query: { id: row.id } })
}

async function onDelete(row: ArticleRow) {
  await ElMessageBox.confirm(`确认删除文章「${row.title}」?`, '提示', { type: 'warning' })
  await articleApi.remove(row.id)
  ElMessage.success('删除成功')
  tableRef.value?.load()
}
</script>
