<template>
  <PageHeader description="先按文件类型导航，再在类型下按分组浏览；私有文件经鉴权加载。" />

  <el-card class="file-center-card" :body-style="{ padding: '0 16px 16px' }">
    <!-- 类型标签是最外层导航:左栏分组及其计数都在它之下,切换标签要连带刷新计数 -->
    <el-tabs :model-value="category" class="kind-tabs" @tab-change="onTabChange">
      <el-tab-pane v-for="opt in categoryOptions" :key="opt.value" :label="opt.label" :name="opt.value" />
    </el-tabs>

    <aside class="file-center__side">
      <ul v-loading="groupsLoading" class="group-list">
        <li
          v-for="node in groupNodes"
          :key="node.key"
          class="group-list__item"
          :class="{ 'is-active': activeKey === node.key }"
          :title="node.name"
          @click="selectGroup(node)"
        >
          <el-icon class="group-list__icon"><component :is="node.icon" /></el-icon>
          <span class="group-list__name">{{ node.name }}</span>
          <span class="group-list__count">{{ node.count }}</span>
          <span v-if="node.groupId" class="group-list__ops">
            <el-button
              v-perm="'system:filegroup:update'"
              :icon="EditPen"
              link
              title="重命名分组"
              @click.stop="renameGroup(node)"
            />
            <el-button
              v-perm="'system:filegroup:delete'"
              :icon="Delete"
              link
              title="删除分组"
              @click.stop="removeGroup(node)"
            />
          </span>
        </li>
      </ul>
      <el-button
        v-perm="'system:filegroup:create'"
        class="group-add"
        :icon="Plus"
        text
        bg
        @click="createGroup"
      >
        新增分组
      </el-button>
    </aside>

    <section class="file-center__main">
      <div class="toolbar">
        <!-- accept 与后端上传白名单保持一致;放宽需服务端评审,不要在这里单方面加类型 -->
        <el-upload
          :show-file-list="false"
          :auto-upload="false"
          :on-change="onFileChange"
          accept=".jpg,.jpeg,.png,.gif,.webp,.pdf,.txt,.csv,.xlsx,.xls,.docx,.zip"
        >
          <el-button v-perm="'system:file:upload'" type="primary" :icon="Upload" :loading="uploading">
            上传
          </el-button>
        </el-upload>
        <el-button
          v-perm="'system:file:delete'"
          :icon="Delete"
          :disabled="!selectedIds.length"
          @click="batchDelete"
        >
          删除
        </el-button>
        <el-button
          v-perm="'system:file:move'"
          :icon="Position"
          :disabled="!selectedIds.length"
          @click="moveOpen = true"
        >
          移动
        </el-button>

        <div class="toolbar__right">
          <el-input
            v-model="originName"
            class="toolbar__search"
            placeholder="搜索文件名"
            clearable
            :prefix-icon="Search"
            @keyup.enter="search"
            @clear="search"
          />
          <FilterActions @search="search" @reset="resetFilters" />
          <el-radio-group v-model="viewMode" class="view-switch">
            <el-radio-button value="grid" title="网格视图">
              <el-icon><Grid /></el-icon>
            </el-radio-button>
            <el-radio-button value="list" title="列表视图">
              <el-icon><List /></el-icon>
            </el-radio-button>
          </el-radio-group>
        </div>
      </div>

      <div class="select-bar">
        <el-checkbox
          :model-value="allSelected"
          :indeterminate="someSelected"
          :disabled="!rows.length"
          @change="(checked: CheckboxValueType) => toggleAll(Boolean(checked))"
        >
          全选
        </el-checkbox>
        <span class="select-bar__hint">已选 {{ selectedIds.length }} 项</span>
      </div>

      <div v-loading="loading" class="file-center__content">
        <ul v-if="viewMode === 'grid'" ref="rootRef" class="file-grid">
          <FileCard
            v-for="row in rows"
            :key="row.id"
            :data-thumb-id="row.id"
            :file="row"
            :thumb-url="thumbUrl(row)"
            :selected="isSelected(row)"
            @download="onDownload"
            @remove="onDelete"
            @preview="onPreview"
            @select="toggleOne"
          />
        </ul>
        <el-table v-else ref="tableRef" :data="rows" row-key="id" @selection-change="onTableSelect">
          <el-table-column type="selection" width="44" />
          <el-table-column label="文件名" min-width="220" show-overflow-tooltip>
            <template #default="{ row }">
              <el-icon class="row-icon"><component :is="fileIcon(row)" /></el-icon>
              <span>{{ row.originName }}</span>
            </template>
          </el-table-column>
          <el-table-column label="类型" width="90">
            <template #default="{ row }">{{ row.ext || '—' }}</template>
          </el-table-column>
          <el-table-column label="大小" width="100">
            <template #default="{ row }">{{ formatSize(row.size) }}</template>
          </el-table-column>
          <el-table-column label="分组" width="140" show-overflow-tooltip>
            <template #default="{ row }">{{ groupName(row.groupId) }}</template>
          </el-table-column>
          <el-table-column prop="uploader" label="上传人" width="120" show-overflow-tooltip />
          <el-table-column label="上传时间" width="170">
            <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="110" fixed="right">
            <template #default="{ row }">
              <el-button :icon="Download" link title="下载" @click="onDownload(row)" />
              <el-button v-perm="'system:file:delete'" :icon="Delete" link title="删除" @click="onDelete(row)" />
            </template>
          </el-table-column>
          <template #empty>
            <el-empty description="暂无文件" :image-size="80" />
          </template>
        </el-table>
        <el-empty v-if="viewMode === 'grid' && !rows.length && !loading" description="暂无文件" :image-size="90" />
      </div>

      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        class="file-pager"
        :total="total"
        :page-sizes="[12, 24, 48, 96]"
        layout="total, sizes, prev, pager, next, jumper"
        @current-change="load"
        @size-change="search"
      />
    </section>
  </el-card>

  <el-dialog v-model="moveOpen" title="移动到分组" width="380px">
    <el-radio-group v-model="moveTarget" class="move-list">
      <el-radio :value="0">未分组</el-radio>
      <el-radio v-for="g in tree.groups" :key="g.id" :value="g.id">{{ g.name }}</el-radio>
    </el-radio-group>
    <p v-if="!tree.groups.length" class="move-list__hint">还没有分组,可先在左侧新增一个。</p>
    <template #footer>
      <el-button @click="moveOpen = false">取消</el-button>
      <el-button type="primary" :loading="moving" @click="confirmMove">
        移动 {{ selectedIds.length }} 个文件
      </el-button>
    </template>
  </el-dialog>

  <el-image-viewer
    v-if="previewUrls.length"
    :url-list="previewUrls"
    :initial-index="previewIndex"
    teleported
    hide-on-click-modal
    @close="closePreview"
  />
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { CheckboxValueType, TabPaneName, TableInstance, UploadFile } from 'element-plus'
import { Delete, Download, EditPen, Grid, List, Plus, Position, Search, Upload } from '@element-plus/icons-vue'
import {
  fileApi,
  fileGroupApi,
  type FileCategory,
  type FileGroupTree,
  type FileListParams,
  type FileRow
} from '../../../api'
import PageHeader from '../../../components/PageHeader.vue'
import FilterActions from '../../../components/FilterActions.vue'
import FileCard from './FileCard.vue'
import { fileIcon, isImageFile, publicSrc } from './fileDisplay'
import { useFileThumbnails } from './useFileThumbnails'
import { formatDateTime, formatSize } from '../../../utils/format'

// 视图选择是个人偏好,存本地;服务端不记录,也不影响数据契约。
type ViewMode = 'grid' | 'list'
const VIEW_MODE_KEY = 'file-view-mode'

const rows = ref<FileRow[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(24)
const loading = ref(false)
const uploading = ref(false)
const originName = ref('')
// 默认落在第一档;重置与初始值共用,避免两处漂移。
const DEFAULT_CATEGORY: FileCategory = 'image'
const category = ref<FileCategory>(DEFAULT_CATEGORY)
const viewMode = ref<ViewMode>(localStorage.getItem(VIEW_MODE_KEY) === 'list' ? 'list' : 'grid')

// 只有三档:图片/视频/文件互斥且穷尽(文件档是前两档的补集),所以不需要"全部类型"这一档。
// 后端仍接受 category 缺省=不筛选,那是接口能力,不在这里做成第四个标签造成与左栏"全部"撞名。
const categoryOptions: { label: string; value: FileCategory }[] = [
  { label: '图片', value: 'image' },
  { label: '视频', value: 'video' },
  { label: '文件', value: 'file' },
]

// ---- 分组左栏 ----
const tree = ref<FileGroupTree>({ groups: [], unfiled: 0, total: 0 })
const groupsLoading = ref(false)
// 'all' 与 'unfiled' 是前端固定渲染的伪节点,不写库:客户环境不该被塞进一条叫"未分组"的数据。
const activeKey = ref('all')

interface GroupNode {
  key: string
  groupId?: number
  name: string
  count: number
  icon: string
}

const groupNodes = computed<GroupNode[]>(() => [
  { key: 'all', name: '全部', count: tree.value.total, icon: 'Files' },
  { key: 'unfiled', groupId: 0, name: '未分组', count: tree.value.unfiled, icon: 'FolderOpened' },
  ...tree.value.groups.map((g) => ({
    key: String(g.id),
    groupId: g.id,
    name: g.name,
    count: g.count ?? 0,
    icon: 'Folder'
  }))
])

// undefined=不传该参数(全部分组),0=未分组。两者语义不同,不能塌成同一个值。
function keyToGroupId(key: string): number | undefined {
  if (key === 'all') return undefined
  if (key === 'unfiled') return 0
  return Number(key)
}

const currentGroupId = computed(() => keyToGroupId(activeKey.value))

function groupName(groupId: number): string {
  if (!groupId) return '未分组'
  return tree.value.groups.find((g) => g.id === groupId)?.name ?? `分组 ${groupId}`
}

async function loadGroups() {
  const seq = ++groupSeq
  groupsLoading.value = true
  try {
    const { data } = await fileGroupApi.list(category.value)
    if (seq !== groupSeq) return // 已有更新的请求在飞,旧类型的计数不得落回左栏
    tree.value = data.data
  } catch {
    /* 失败提示由统一拦截器给出 */
  } finally {
    if (seq === groupSeq) groupsLoading.value = false
  }
}

// 类型标签是最高层导航:换标签时列表与左栏计数一起刷新。
// 不重置当前分组 —— 左栏会显示该组在新类型下的数量,为 0 就看得见是 0,比把用户挪走更好懂。
async function onTabChange(name: TabPaneName) {
  category.value = String(name) as FileCategory
  await Promise.all([search(), loadGroups()])
}

function selectGroup(node: GroupNode) {
  if (activeKey.value === node.key) return
  activeKey.value = node.key
  return search()
}

// promptGroupName 复用一次输入框;返回 null 表示用户取消。
async function promptGroupName(title: string, initial = ''): Promise<string | null> {
  try {
    const { value } = await ElMessageBox.prompt('分组名最长 64 个字符,且不能与已有分组同名', title, {
      inputValue: initial,
      inputValidator: (v: string) => (v?.trim() ? true : '请输入分组名')
    })
    return value.trim()
  } catch {
    return null
  }
}

async function createGroup() {
  const name = await promptGroupName('新增分组')
  if (!name) return
  try {
    await fileGroupApi.create(name)
    ElMessage.success('分组已创建')
    await loadGroups()
  } catch {
    /* 同名等冲突由拦截器提示 */
  }
}

async function renameGroup(node: GroupNode) {
  const name = await promptGroupName('重命名分组', node.name)
  if (!name || name === node.name) return
  try {
    await fileGroupApi.update(node.groupId as number, name)
    ElMessage.success('分组已重命名')
    await loadGroups()
  } catch {
    /* 由拦截器提示 */
  }
}

async function removeGroup(node: GroupNode) {
  try {
    await ElMessageBox.confirm(`确认删除空分组「${node.name}」?分组下的文件不会被删除。`, '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await fileGroupApi.remove(node.groupId as number)
    ElMessage.success('分组已删除')
    if (activeKey.value === node.key) activeKey.value = 'all'
    await Promise.all([loadGroups(), load()])
  } catch {
    /* 非空分组会返回冲突,由拦截器提示 */
  }
}

// ---- 列表 ----
const {
  rootRef,
  state,
  setPreviewing
} = useFileThumbnails(rows, computed(() => viewMode.value === 'grid'))

// 快速连切标签/翻页时,先发出的请求可能后返回。各自一个自增序号,只认最新那次,
// 否则旧类型的文件与计数会覆盖到新视图上。
let listSeq = 0
let groupSeq = 0

async function load() {
  const seq = ++listSeq
  loading.value = true
  try {
    const params: FileListParams = {
      page: page.value,
      pageSize: pageSize.value,
      originName: originName.value || undefined,
      category: category.value,
      groupId: currentGroupId.value
    }
    const { data } = await fileApi.list(params)
    if (seq !== listSeq) return
    rows.value = data.data.list
    total.value = data.data.total
    clearSelection()
  } finally {
    if (seq === listSeq) loading.value = false
  }
}

// 筛选条件变化都必须回到第一页,否则会停在旧页码看到空白页
function search() {
  page.value = 1
  return load()
}

async function resetFilters() {
  originName.value = ''
  category.value = DEFAULT_CATEGORY
  activeKey.value = 'all'
  await Promise.all([search(), loadGroups()])
}

function thumbUrl(row: FileRow): string {
  if (!isImageFile(row)) return ''
  if (row.isPublic) return publicSrc(row)
  return state[row.id]?.status === 'ready' ? state[row.id].url : ''
}

// ---- 多选:网格与列表共用同一份 selectedIds ----
const selectedIds = ref<number[]>([])
const tableRef = ref<TableInstance>()

// 视图偏好落本地;切到列表时把手工勾选的 id 回填到表格选中态,
// 否则会看到"顶部计数还在、勾却没打上"的错位状态。
watch(viewMode, async (mode) => {
  localStorage.setItem(VIEW_MODE_KEY, mode)
  if (mode !== 'list' || !selectedIds.value.length) return
  await nextTick()
  const keep = new Set(selectedIds.value)
  for (const row of rows.value) {
    if (keep.has(row.id)) tableRef.value?.toggleRowSelection(row, true)
  }
})

function isSelected(row: FileRow): boolean {
  return selectedIds.value.includes(row.id)
}

function clearSelection() {
  selectedIds.value = []
  tableRef.value?.clearSelection()
}

function toggleOne(row: FileRow) {
  const i = selectedIds.value.indexOf(row.id)
  if (i >= 0) selectedIds.value.splice(i, 1)
  else selectedIds.value.push(row.id)
}

const allSelected = computed(() => rows.value.length > 0 && rows.value.every(isSelected))
const someSelected = computed(() => selectedIds.value.length > 0 && !allSelected.value)

function onTableSelect(selection: FileRow[]) {
  selectedIds.value = selection.map((r) => r.id)
}

// 列表视图的行勾选由 el-table 持有,全选/清空都必须镜像过去,
// 否则顶部计数与表格实际选中会分叉。
function toggleAll(checked: boolean) {
  if (!checked) {
    clearSelection()
    return
  }
  selectedIds.value = rows.value.map((r) => r.id)
  if (viewMode.value === 'list') {
    void nextTick(() => {
      for (const row of rows.value) tableRef.value?.toggleRowSelection(row, true)
    })
  }
}

// ---- 批量操作 ----
const moveOpen = ref(false)
const moving = ref(false)
const moveTarget = ref(0)

async function confirmMove() {
  moving.value = true
  try {
    await fileApi.move(selectedIds.value, moveTarget.value)
    ElMessage.success('已移动')
    moveOpen.value = false
    await Promise.all([load(), loadGroups()])
  } catch {
    /* 由拦截器提示 */
  } finally {
    moving.value = false
  }
}

async function batchDelete() {
  const ids = [...selectedIds.value]
  if (!ids.length) return
  try {
    await ElMessageBox.confirm(`确认删除选中的 ${ids.length} 个文件?该操作不可撤销。`, '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await fileApi.batchRemove(ids)
    ElMessage.success('删除成功')
    await afterMutation(ids.length)
  } catch {
    /* 由拦截器提示 */
  }
}

// 删掉当前页最后几条时回退一页,避免停在空页上以为没有数据了
async function afterMutation(removedCount: number) {
  if (rows.value.length === removedCount && page.value > 1) page.value--
  await Promise.all([load(), loadGroups()])
}

// ---- 大图预览:复用缩略图已拿到的地址,不重复下载 ----
const previewUrls = ref<string[]>([])
const previewIndex = ref(0)

function onPreview(row: FileRow) {
  const src = thumbUrl(row)
  if (!src) {
    ElMessage.info('该图片尚未加载出缩略图(可能过大或加载失败),可点下载查看原图')
    return
  }
  const urls = rows.value.filter(isImageFile).map(thumbUrl).filter(Boolean)
  previewUrls.value = urls
  previewIndex.value = Math.max(urls.indexOf(src), 0)
  // 预览期间这张 objectURL 不能被翻页回收
  setPreviewing(row.id)
}

function closePreview() {
  previewUrls.value = []
  setPreviewing(null)
}

// ---- 上传 / 下载 / 单条删除 ----
async function onFileChange(uploadFile: UploadFile) {
  // el-upload 多选会连续触发 on-change,单飞避免并发上传互相覆盖 loading
  if (!uploadFile.raw || uploading.value) return
  uploading.value = true
  try {
    // 落在当前选中的分组;"全部"视图没有明确归属,按未分组处理
    await fileApi.upload(uploadFile.raw, false, currentGroupId.value ?? 0)
    ElMessage.success('上传成功')
    await Promise.all([load(), loadGroups()])
  } catch {
    /* 失败提示由统一拦截器给出 */
  } finally {
    uploading.value = false
  }
}

function triggerDownload(href: string, filename: string) {
  const a = document.createElement('a')
  a.href = href
  a.download = filename
  a.click()
}

async function onDownload(row: FileRow) {
  if (row.isPublic) {
    triggerDownload(publicSrc(row), row.originName)
    return
  }
  // 私有文件:<a href> 直链带不了 Authorization,会被 401 拦下,只能经鉴权层取字节
  try {
    const { data } = await fileApi.fetchBlob(row.id, false)
    const url = URL.createObjectURL(data)
    triggerDownload(url, row.originName)
    // 立刻回收会让部分浏览器来不及发起下载
    setTimeout(() => URL.revokeObjectURL(url), 1000)
  } catch {
    /* 失败提示由统一拦截器给出 */
  }
}

async function onDelete(row: FileRow) {
  try {
    await ElMessageBox.confirm(`确认删除「${row.originName}」?`, '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await fileApi.remove(row.id)
    ElMessage.success('删除成功')
    await afterMutation(1)
  } catch {
    /* 由拦截器提示 */
  }
}

onMounted(() => {
  void loadGroups()
  void load()
})
</script>

<style scoped>
/* 类型标签横跨整宽,其下才是"分组栏 + 内容区"两列 —— 层级与后端一致 */
.file-center-card :deep(.el-card__body) {
  display: grid;
  grid-template-columns: 190px minmax(0, 1fr);
  align-items: start;
  column-gap: 16px;
}

.kind-tabs {
  grid-column: 1 / -1;
}

.kind-tabs :deep(.el-tabs__header) {
  margin-bottom: 0;
}

.file-center__side {
  padding: 8px 12px 8px 0;
  border-right: 1px solid var(--el-border-color-lighter);
}

.file-center__main {
  min-width: 0;
  padding-top: 12px;
}

.group-list {
  margin: 0;
  padding: 0;
  list-style: none;
}

.group-list__item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 8px;
  border-radius: 6px;
  font-size: 13px;
  color: var(--el-text-color-regular);
  cursor: pointer;
}

.group-list__item:hover {
  background: var(--el-fill-color-light);
}

.group-list__item.is-active {
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  font-weight: 600;
}

.group-list__icon {
  flex: none;
}

.group-list__name {
  flex: 1;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.group-list__count {
  flex: none;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

/* 操作按钮只在 hover 当前行时出现,否则左栏会被图标挤成一团 */
.group-list__ops {
  display: none;
  flex: none;
}

.group-list__item:hover .group-list__ops {
  display: inline-flex;
}

.group-list__item:hover .group-list__count {
  display: none;
}

.group-add {
  width: 100%;
  margin-top: 6px;
}

.toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.toolbar__right {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-left: auto;
}

.toolbar__search {
  width: 200px;
}

.select-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 10px 0 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.file-center__content {
  min-height: 260px;
}

.file-grid {
  display: grid;
  /* auto-fill:条目不满一行时保持固定卡片宽度,不会被拉成巨卡 */
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 14px;
  margin: 0;
  padding: 0;
}

.row-icon {
  margin-right: 6px;
  vertical-align: -2px;
  color: var(--el-text-color-secondary);
}

/* 分页在右下角,与参考图一致 */
.file-pager {
  margin-top: 14px;
  justify-content: flex-end;
}

.move-list {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
}

.move-list__hint {
  margin: 8px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
