<template>
  <PageHeader description="提供用户添加、编辑、删除功能，超管不可修改。">
    <template #extra>
      <el-button v-perm="'system:user:create'" type="primary" @click="openCreate">新建用户</el-button>
      <el-button v-perm="'system:user:export'" @click="onExport">导出 Excel</el-button>
    </template>
  </PageHeader>
  <PaginatedTable ref="tableRef" :fetch="userApi.list" :query="filters">
    <template #toolbar>
      <el-input v-model="filters.username" placeholder="用户名" clearable style="width: 180px" @keyup.enter="tableRef?.search()" />
      <el-select v-model="filters.status" placeholder="状态" clearable style="width: 120px">
        <el-option label="启用" :value="1" />
        <el-option label="停用" :value="2" />
      </el-select>
      <FilterActions @search="tableRef?.search()" @reset="resetFilters" />
    </template>

    <el-table-column prop="id" label="ID" width="70" />
    <el-table-column label="头像" width="70" align="center">
      <template #default="{ row }">
        <el-avatar v-if="row.avatar" :src="row.avatar" :size="32" />
        <el-avatar v-else :size="32">{{ (row.nickname || row.username).slice(0, 1) }}</el-avatar>
      </template>
    </el-table-column>
    <el-table-column prop="username" label="用户名" min-width="110" />
    <el-table-column prop="nickname" label="昵称" min-width="100" show-overflow-tooltip />
    <el-table-column prop="email" label="邮箱" min-width="150" show-overflow-tooltip />
    <el-table-column prop="phone" label="手机号" width="130" />
    <el-table-column label="角色" min-width="150" align="center">
      <template #default="{ row }">
        <div class="role-tags">
          <el-tag v-if="row.super" type="danger" size="small">超级管理员</el-tag>
          <template v-else-if="roleNames(row).length">
            <el-tag
              v-for="r in roleNames(row)"
              :key="r.id"
              size="small"
              :type="r.status === 1 ? 'info' : 'warning'"
            >{{ r.status === 1 ? r.name : `${r.name}(停用)` }}</el-tag>
          </template>
          <span v-else>-</span>
        </div>
      </template>
    </el-table-column>
    <el-table-column label="备注" min-width="140" show-overflow-tooltip>
      <template #default="{ row }">{{ row.remark || '-' }}</template>
    </el-table-column>
    <el-table-column label="状态" width="90" align="center">
      <template #default="{ row }">
        <el-switch
          :model-value="row.status === 1"
          :disabled="row.super || !auth.hasPerm('system:user:update')"
          :before-change="() => confirmToggleStatus(row)"
          @change="() => toggleStatus(row)"
        />
      </template>
    </el-table-column>
    <el-table-column label="最后登录" width="150">
      <template #default="{ row }">{{ formatDateTime(row.lastLoginAt) }}</template>
    </el-table-column>
    <el-table-column label="操作" width="130" fixed="right" align="center">
      <template #default="{ row }">
        <el-button v-perm="'system:user:update'" link type="primary" @click="openEdit(row)">编辑</el-button>
        <el-dropdown trigger="click" @command="(cmd: string) => onRowCommand(cmd, row)">
          <el-button link type="primary">
            更多<el-icon class="more-arrow"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item v-perm="'system:user:reset-password'" command="resetPassword" :disabled="row.super">
                重置密码
              </el-dropdown-item>
              <el-dropdown-item v-perm="'system:user:assign-role'" command="assignRoles" :disabled="row.super">
                分配角色
              </el-dropdown-item>
              <el-dropdown-item v-perm="'system:user:delete'" command="delete" :disabled="row.super" divided>
                <span class="danger-item">删除</span>
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </template>
    </el-table-column>
  </PaginatedTable>

  <!-- 新建/编辑 -->
  <el-dialog v-model="editVisible" :title="editForm.id ? '编辑用户' : '新建用户'" width="480px" @closed="clearPendingAvatar">
    <el-form :model="editForm" label-width="80px">
      <el-form-item label="头像" class="avatar-form-item">
        <div class="avatar-editor">
          <div class="avatar-wrapper">
            <el-avatar v-if="avatarPreview" :src="avatarPreview" :size="52" class="user-avatar-preview" />
            <el-avatar v-else :size="52" class="user-avatar-preview user-avatar-placeholder">
              {{ (editForm.nickname || editForm.username || '用').slice(0, 1).toUpperCase() }}
            </el-avatar>
          </div>
          <div class="avatar-action-box">
            <el-upload
              v-if="editForm.id"
              :show-file-list="false"
              accept=".jpg,.jpeg,.png,.gif,.webp"
              :http-request="onUploadAvatar"
            >
              <el-button size="small" type="primary" plain>上传新头像</el-button>
            </el-upload>
            <el-upload
              v-else
              :show-file-list="false"
              :auto-upload="false"
              accept=".jpg,.jpeg,.png,.gif,.webp"
              :on-change="onSelectAvatar"
            >
              <el-button size="small" type="primary" plain>选择头像</el-button>
            </el-upload>
            <span class="avatar-tip">支持 JPG/PNG/WEBP，不超过 2MB</span>
          </div>
        </div>
      </el-form-item>
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
      <el-form-item label="备注">
        <el-input v-model="editForm.remark" type="textarea" :rows="3" maxlength="255" show-word-limit />
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
      <el-option v-for="r in enabledRoles" :key="r.id" :label="r.name" :value="r.id" />
    </el-select>
    <template #footer>
      <el-button @click="rolesVisible = false">取消</el-button>
      <el-button type="primary" @click="saveRoles">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox, type UploadFile } from 'element-plus'
import { ArrowDown } from '@element-plus/icons-vue'
import { userApi, roleApi, type UserItem, type RoleItem } from '../../../api'
import { useAuthStore } from '../../../stores/auth'
import { formatDateTime } from '../../../utils/format'
import PaginatedTable from '../../../components/PaginatedTable.vue'
import PageHeader from '../../../components/PageHeader.vue'
import FilterActions from '../../../components/FilterActions.vue'
import type { PaginatedTableHandle } from '../../../components/paginated-table'

const tableRef = ref<PaginatedTableHandle | undefined>()
const auth = useAuthStore()
const filters = reactive<{ username: string; status?: number }>({ username: '', status: undefined })

function resetFilters() {
  filters.username = ''
  filters.status = undefined
  tableRef.value?.search()
}

const editVisible = ref(false)
const editForm = reactive({ id: 0, username: '', password: '', nickname: '', email: '', phone: '', status: 1, remark: '', avatar: '' })
const pendingAvatar = ref<File>()
const pendingAvatarUrl = ref('')
const avatarPreview = computed(() => pendingAvatarUrl.value || editForm.avatar)

function clearPendingAvatar() {
  if (pendingAvatarUrl.value) URL.revokeObjectURL(pendingAvatarUrl.value)
  pendingAvatar.value = undefined
  pendingAvatarUrl.value = ''
}

const rolesVisible = ref(false)
const selectedRoleIds = ref<number[]>([])
const allRoles = ref<RoleItem[]>([])
let rolesTarget = 0

// 角色字典:进页面拉一次,供列表角色列与分配角色弹窗共用(含停用角色,列表列需要展示其名)
onMounted(loadRoles)
async function loadRoles() {
  const { data } = await roleApi.list({ page: 1, pageSize: 100 })
  allRoles.value = data.data.list
}

// 行的角色标签数据:id→角色映射;停用角色标出「(停用)」并以警告色提示
function roleNames(row: UserItem) {
  return allRoles.value.filter((r) => row.roleIds.includes(r.id))
}
// 分配角色弹窗只允许勾选启用中的角色
const enabledRoles = computed(() => allRoles.value.filter((r) => r.status === 1))

function openCreate() {
  clearPendingAvatar()
  Object.assign(editForm, { id: 0, username: '', password: '', nickname: '', email: '', phone: '', status: 1, remark: '', avatar: '' })
  editVisible.value = true
}

function openEdit(row: UserItem) {
  clearPendingAvatar()
  Object.assign(editForm, {
    id: row.id, username: row.username, password: '', nickname: row.nickname,
    email: row.email, phone: row.phone, status: row.status, remark: row.remark, avatar: row.avatar
  })
  editVisible.value = true
}

async function save() {
  if (editForm.id) {
    await userApi.update(editForm.id, {
      nickname: editForm.nickname, email: editForm.email, phone: editForm.phone, status: editForm.status, remark: editForm.remark
    })
  } else {
    const { data } = await userApi.create({
      username: editForm.username, password: editForm.password, nickname: editForm.nickname,
      email: editForm.email, phone: editForm.phone, status: editForm.status, remark: editForm.remark
    })
    if (pendingAvatar.value) {
      const fd = new FormData()
      fd.append('file', pendingAvatar.value)
      try {
        await userApi.uploadAvatar(data.data.id, fd)
      } catch {
        ElMessage.warning('用户已创建，但头像上传失败；可在编辑用户中重试')
        editVisible.value = false
        tableRef.value?.load()
        return
      }
    }
  }
  ElMessage.success('保存成功')
  editVisible.value = false
  clearPendingAvatar()
  tableRef.value?.load()
}

function onSelectAvatar(file: UploadFile) {
  if (!file.raw) return
  clearPendingAvatar()
  pendingAvatar.value = file.raw
  pendingAvatarUrl.value = URL.createObjectURL(file.raw)
}

// 管理端替用户上传头像:走后端与个人中心相同的校验链路(类型/大小/真实 MIME),
// 成功后更新表单预览并刷新列表;新建时由 save 在创建成功后绑定暂存头像。
async function onUploadAvatar(opts: { file: File }) {
  const fd = new FormData()
  fd.append('file', opts.file)
  const { data } = await userApi.uploadAvatar(editForm.id, fd)
  editForm.avatar = data.data.avatar
  ElMessage.success('头像已更新')
  tableRef.value?.load()
}

async function onDelete(row: UserItem) {
  await ElMessageBox.confirm(`确认删除用户「${row.username}」?`, '提示', { type: 'warning' })
  await userApi.remove(row.id)
  ElMessage.success('删除成功')
  tableRef.value?.load()
}

// before-change 拦截停用方向:停用会立即失效该用户已签发凭据,需二次确认;启用直接放行
function confirmToggleStatus(row: UserItem) {
  if (row.status !== 1) return Promise.resolve(true)
  return ElMessageBox.confirm(`确认停用用户「${row.username}」?停用后其登录凭据将立即失效。`, '提示', { type: 'warning' })
}

async function toggleStatus(row: UserItem) {
  await userApi.setStatus(row.id, row.status === 1 ? 2 : 1)
  ElMessage.success('操作成功')
  tableRef.value?.load()
}

// 「更多」下拉动作分发;动作实现与原行内按钮一致
function onRowCommand(cmd: string, row: UserItem) {
  switch (cmd) {
    case 'resetPassword':
      onResetPassword(row)
      break
    case 'assignRoles':
      openRoles(row)
      break
    case 'delete':
      onDelete(row)
      break
  }
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

<style scoped>
.more-arrow {
  margin-left: 2px;
  font-size: 12px;
}

.danger-item {
  color: var(--el-color-danger);
}

.role-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  justify-content: center;
}

.avatar-editor {
  display: flex;
  align-items: center;
  gap: 16px;
}

.avatar-wrapper {
  position: relative;
  border-radius: 50%;
  padding: 2px;
  background: #ffffff;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08), 0 0 0 1px rgba(0, 0, 0, 0.06);
}

.user-avatar-preview {
  display: block;
}

.user-avatar-placeholder {
  background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
  color: #ffffff;
  font-weight: 600;
  font-size: 18px;
}

.avatar-action-box {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
}

/* 默认表单标签按 36px 行高从顶部对齐；头像行高度更高时需与内容垂直居中。 */
.avatar-form-item {
  align-items: center;
}

.avatar-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
