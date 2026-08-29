<template>
  <PageHeader description="字典是「一个类型下多条子项」的枚举集,业务模块按字典键读取启用子项;内置字典可维护子项,不可改键名或删除。" />
  <el-row :gutter="10">
    <el-col :span="7">
      <el-card v-loading="loading" class="type-card">
        <template #header>
          <div class="panel-header">
            <span>字典类型</span>
            <span class="panel-actions">
              <el-input
                v-model="typeKeyword"
                placeholder="字典名 / 字典键"
                clearable
                style="width: 170px"
                size="small"
              />
              <el-button v-perm="'system:dict:create'" type="primary" size="small" @click="openCreateType">新建字典</el-button>
            </span>
          </div>
        </template>
        <el-empty v-if="!filteredTypes.length" description="暂无字典" :image-size="60" />
        <div v-else class="type-list">
          <div
            v-for="t in filteredTypes"
            :key="t.id"
            class="type-item"
            :class="{ active: currentType?.id === t.id, 'no-actions': !canManageDict }"
            @click="onSelectType(t)"
          >
            <span class="type-name">
              {{ t.name }}
              <el-tag v-if="t.builtin" type="warning" size="small">内置</el-tag>
            </span>
            <span class="type-meta">
              <span class="type-key">{{ t.key }}</span>
              <span v-if="t.itemCount" class="type-count">{{ t.itemCount }}</span>
              <span class="type-actions">
                <el-button
                  v-perm="'system:dict:update'"
                  size="small"
                  :icon="Edit"
                  title="编辑"
                  @click.stop="openEditType(t)"
                />
                <el-button
                  v-perm="'system:dict:delete'"
                  class="delete"
                  size="small"
                  :icon="Delete"
                  title="删除"
                  :disabled="t.builtin"
                  @click.stop="onDeleteType(t)"
                />
              </span>
            </span>
          </div>
        </div>
      </el-card>
    </el-col>
    <el-col :span="17">
      <el-card class="items-card">
        <template #header>
          <div class="panel-header">
            <span>子项{{ currentType ? ` —— ${currentType.name}` : '(先选择左侧字典)' }}</span>
            <el-button
              v-perm="'system:dict:create'"
              type="primary"
              size="small"
              :disabled="!currentType"
              @click="openCreateItem"
            >新建子项</el-button>
          </div>
        </template>
        <el-table v-loading="itemsLoading" :data="items">
          <el-table-column prop="value" label="数据值" min-width="90" align="center" />
          <el-table-column prop="label" label="标签名" min-width="90" align="center" />
          <el-table-column label="描述" min-width="140" align="center" show-overflow-tooltip>
            <template #default="{ row }">{{ row.description || '-' }}</template>
          </el-table-column>
          <el-table-column prop="sort" label="排序" width="70" align="center" />
          <el-table-column label="标签类型" width="110" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.tagType" :type="row.tagType" size="small">{{ tagTypeLabel(row.tagType) }}</el-tag>
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column label="备注" min-width="120" align="center" show-overflow-tooltip>
            <template #default="{ row }">{{ row.remark || '-' }}</template>
          </el-table-column>
          <el-table-column label="创建时间" width="150" align="center">
            <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right" align="center">
            <template #default="{ row }">
              <el-button v-perm="'system:dict:update'" link type="primary" @click="openEditItem(row)">编辑</el-button>
              <el-button v-perm="'system:dict:delete'" link type="danger" @click="onDeleteItem(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </el-col>
  </el-row>

  <el-dialog v-model="typeVisible" :title="typeForm.id ? '编辑字典' : '新建字典'" width="440px">
    <el-form :model="typeForm" label-width="70px">
      <el-form-item label="字典名" required><el-input v-model="typeForm.name" maxlength="64" /></el-form-item>
      <el-form-item label="字典键" required>
        <el-input v-model="typeForm.key" maxlength="64" :disabled="isBuiltinType" placeholder="小写字母开头,可用 . - _" />
      </el-form-item>
      <el-form-item label="备注"><el-input v-model="typeForm.remark" maxlength="255" /></el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="typeVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="saveType">保存</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="itemVisible" :title="itemForm.id ? '编辑子项' : '新建子项'" width="480px">
    <el-form :model="itemForm" label-width="70px">
      <el-form-item label="标签名" required><el-input v-model="itemForm.label" maxlength="64" /></el-form-item>
      <el-form-item label="数据值" required><el-input v-model="itemForm.value" maxlength="128" /></el-form-item>
      <el-form-item label="描述"><el-input v-model="itemForm.description" type="textarea" :rows="2" maxlength="255" /></el-form-item>
      <el-form-item label="排序">
        <el-input-number v-model="itemForm.sort" :min="0" controls-position="right" style="width: 100%" />
      </el-form-item>
      <el-form-item label="标签类型">
        <el-select v-model="itemForm.tagType" placeholder="请选择标签类型" style="width: 100%">
          <el-option label="默认" value="" />
          <el-option v-for="t in TAG_TYPES" :key="t.value" :label="t.label" :value="t.value">
            <el-tag :type="t.value" effect="light">{{ t.label }}</el-tag>
          </el-option>
        </el-select>
      </el-form-item>
      <el-form-item label="状态">
        <el-radio-group v-model="itemForm.status">
          <el-radio :value="1">启用</el-radio>
          <el-radio :value="2">停用</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="备注"><el-input v-model="itemForm.remark" maxlength="255" /></el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="itemVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="saveItem">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Edit } from '@element-plus/icons-vue'
import { dictTypeApi, type DictItem, type DictType } from '../../../api'
import { useAuthStore } from '../../../stores/auth'
import PageHeader from '../../../components/PageHeader.vue'
import { formatDateTime } from '../../../utils/format'

// 字典类型是少量枚举,不分页;左类型右子项,选中类型后右侧实时拉取其子项。
const types = ref<DictType[]>([])
const typeKeyword = ref('')
const loading = ref(false)

const filteredTypes = computed(() => {
  const kw = typeKeyword.value.trim().toLowerCase()
  if (!kw) return types.value
  return types.value.filter((t) => t.name.toLowerCase().includes(kw) || t.key.toLowerCase().includes(kw))
})

async function loadTypes(keepSelected = true) {
  loading.value = true
  try {
    const { data } = await dictTypeApi.list()
    types.value = data.data
    if (keepSelected && currentType.value) {
      currentType.value = types.value.find((t) => t.id === currentType.value?.id) ?? null
    }
  } finally {
    loading.value = false
  }
}

const currentType = ref<DictType | null>(null)
// 悬停时 key 让位给操作按钮;完全无维护权限的角色保持常显 key,避免悬停出现空白
const auth = useAuthStore()
const canManageDict = computed(() => auth.hasPerm('system:dict:update') || auth.hasPerm('system:dict:delete'))
const items = ref<DictItem[]>([])
const itemsLoading = ref(false)

async function onSelectType(row: DictType | null) {
  currentType.value = row
  if (!row) {
    items.value = []
    return
  }
  itemsLoading.value = true
  try {
    const { data } = await dictTypeApi.items(row.id)
    items.value = data.data
  } finally {
    itemsLoading.value = false
  }
}

// ---- 字典类型维护 ----

const typeVisible = ref(false)
const saving = ref(false)
const typeForm = reactive({ id: 0, name: '', key: '', remark: '', builtin: false })
const isBuiltinType = computed(() => typeForm.id > 0 && typeForm.builtin)

function openCreateType() {
  Object.assign(typeForm, { id: 0, name: '', key: '', remark: '', builtin: false })
  typeVisible.value = true
}

function openEditType(row: DictType) {
  Object.assign(typeForm, { id: row.id, name: row.name, key: row.key, remark: row.remark, builtin: row.builtin })
  typeVisible.value = true
}

async function saveType() {
  if (!typeForm.name.trim()) {
    ElMessage.warning('请输入字典名')
    return
  }
  if (!typeForm.key.trim()) {
    ElMessage.warning('请输入字典键')
    return
  }
  saving.value = true
  try {
    const payload = { name: typeForm.name.trim(), key: typeForm.key.trim(), remark: typeForm.remark.trim() }
    if (typeForm.id) {
      await dictTypeApi.update(typeForm.id, payload)
    } else {
      await dictTypeApi.create(payload)
    }
    ElMessage.success('保存成功')
    typeVisible.value = false
    await loadTypes()
  } finally {
    saving.value = false
  }
}

async function onDeleteType(row: DictType) {
  await ElMessageBox.confirm(
    row.itemCount ? `字典「${row.name}」下有 ${row.itemCount} 个子项,删除前需先清空子项。确认删除?` : `确认删除字典「${row.name}」?`,
    '提示',
    { type: 'warning' }
  )
  await dictTypeApi.remove(row.id)
  ElMessage.success('删除成功')
  if (currentType.value?.id === row.id) {
    currentType.value = null
    items.value = []
  }
  await loadTypes()
}

// ---- 子项维护 ----

// 与后端 tagTypes 白名单一致;空串表示默认(不配标签),选项按参考样式用彩色 el-tag 呈现
const TAG_TYPES = [
  { label: '主要 Primary', value: 'primary' },
  { label: '成功 Success', value: 'success' },
  { label: '信息 Info', value: 'info' },
  { label: '警告 Warning', value: 'warning' },
  { label: '危险 Danger', value: 'danger' }
] as const

function tagTypeLabel(value: string) {
  return TAG_TYPES.find((t) => t.value === value)?.label ?? value
}

const itemVisible = ref(false)
const itemForm = reactive({
  id: 0, typeId: 0, label: '', description: '', value: '', sort: 0, tagType: '', status: 1, remark: ''
})

function openCreateItem() {
  if (!currentType.value) return
  Object.assign(itemForm, {
    id: 0, typeId: currentType.value.id, label: '', description: '', value: '',
    sort: 0, tagType: '', status: 1, remark: ''
  })
  itemVisible.value = true
}

function openEditItem(row: DictItem) {
  Object.assign(itemForm, {
    id: row.id, typeId: row.typeId, label: row.label, description: row.description,
    value: row.value, sort: row.sort, tagType: row.tagType, status: row.status, remark: row.remark
  })
  itemVisible.value = true
}

async function saveItem() {
  if (!itemForm.label.trim()) {
    ElMessage.warning('请输入标签名')
    return
  }
  if (!itemForm.value.trim()) {
    ElMessage.warning('请输入数据值')
    return
  }
  saving.value = true
  try {
    const payload = {
      label: itemForm.label.trim(),
      description: itemForm.description.trim(),
      value: itemForm.value.trim(),
      sort: itemForm.sort,
      tagType: itemForm.tagType,
      status: itemForm.status,
      remark: itemForm.remark.trim()
    }
    if (itemForm.id) {
      await dictTypeApi.updateItem(itemForm.id, payload)
    } else {
      await dictTypeApi.createItem(itemForm.typeId, payload)
    }
    ElMessage.success('保存成功')
    itemVisible.value = false
    if (currentType.value) {
      await onSelectType(currentType.value)
    }
    await loadTypes()
  } finally {
    saving.value = false
  }
}

async function onDeleteItem(row: DictItem) {
  await ElMessageBox.confirm(`确认删除子项「${row.label}」?`, '提示', { type: 'warning' })
  await dictTypeApi.removeItem(row.id)
  ElMessage.success('删除成功')
  if (currentType.value) {
    await onSelectType(currentType.value)
  }
  await loadTypes()
}

onMounted(() => loadTypes(false))
</script>

<style scoped>
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.panel-actions {
  display: inline-flex;
  align-items: center;
  gap: 10px;
}

/* 左卡片定高:按 顶部导航60 + 内容区padding20 + PageHeader卡片62 + 下边距10 预留,
   使卡片底部与视口内容区底部对齐;头部固定,列表区独立滚动 */
.type-card {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 152px);
  min-height: 360px;
}

.type-card :deep(.el-card__header) {
  flex-shrink: 0;
}

.type-card :deep(.el-card__body) {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

/* 两张卡片的标题栏压矮,与紧凑布局匹配 */
.type-card :deep(.el-card__header),
.items-card :deep(.el-card__header) {
  padding: 10px 16px;
}

/* 左侧类型轻列表:单行紧凑,对齐侧边栏的浅色圆角选中风格 */
.type-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.type-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 9px 10px;
  border-radius: 6px;
  border: 1px solid transparent;
  cursor: pointer;
}

.type-item:hover {
  background: var(--el-fill-color-light);
}

.type-item.active {
  background: var(--el-color-primary-light-9);
  border-color: var(--el-color-primary-light-7);
}

.type-name {
  font-weight: 500;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.type-meta {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.type-key {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-family: 'SF Mono', Menlo, Consolas, monospace;
}

.type-count {
  min-width: 16px;
  padding: 0 5px;
  border-radius: 8px;
  background: var(--el-fill-color);
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 16px;
  text-align: center;
}

/* 悬停时 key/计数隐去,原位浮出描边小方块操作按钮 */
.type-actions {
  display: none;
  align-items: center;
  gap: 6px;
}

.type-item:hover .type-key,
.type-item:not(.no-actions):hover .type-count {
  display: none;
}

.type-item:not(.no-actions):hover .type-actions {
  display: inline-flex;
}

.type-actions .el-button {
  padding: 5px 6px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  background: var(--el-bg-color);
  color: var(--el-text-color-regular);
}

.type-actions .el-button + .el-button {
  margin-left: 0;
}

.type-actions .el-button:hover {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary-light-5);
  background: var(--el-bg-color);
}

.type-actions .el-button.delete:hover {
  color: var(--el-color-danger);
  border-color: var(--el-color-danger-light-5);
}
</style>
