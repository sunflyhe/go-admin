<template>
  <el-drawer v-model="open" title="个人信息" size="560px" :close-on-click-modal="true">
    <el-tabs v-model="tab">
      <el-tab-pane name="basic">
        <template #label>
          <span class="tab-label"><el-icon><User /></el-icon>基本信息</span>
        </template>

        <div class="drawer-body">
          <el-upload
            class="avatar-upload"
            :auto-upload="false"
            :show-file-list="false"
            :on-change="onAvatarChange"
            accept=".jpg,.jpeg,.png,.gif,.webp"
          >
            <div class="avatar-circle" :class="{ 'is-loading': uploading }">
              <img v-if="user?.avatar" :src="user.avatar" :alt="user.nickname || '头像'" class="avatar-circle__img" />
              <div v-else class="avatar-circle__empty">
                <el-icon :size="20"><Avatar /></el-icon>
                <span>请上传头像</span>
              </div>
            </div>
          </el-upload>

          <el-form label-position="left" label-width="80px" class="drawer-form">
            <el-form-item label="用户名">
              <el-input :model-value="user?.username" disabled />
            </el-form-item>
            <el-form-item label="手机">
              <el-input v-model="form.phone" maxlength="32" placeholder="手机号" />
            </el-form-item>
            <el-form-item label="邮箱">
              <el-input v-model="form.email" maxlength="128" placeholder="邮箱" />
            </el-form-item>
            <el-form-item label="昵称">
              <el-input v-model="form.nickname" maxlength="64" placeholder="昵称" />
            </el-form-item>
            <el-form-item label="个人签名">
              <el-input v-model="form.signature" type="textarea" :rows="2" maxlength="255" placeholder="个人签名" />
            </el-form-item>
            <el-form-item label-width="0" class="form-actions">
              <el-button type="primary" :loading="savingProfile" @click="saveProfile">更新个人信息</el-button>
            </el-form-item>
          </el-form>
        </div>
      </el-tab-pane>

      <el-tab-pane name="security">
        <template #label>
          <span class="tab-label"><el-icon><Lock /></el-icon>安全信息</span>
        </template>

        <div class="drawer-body">
          <el-form label-position="left" label-width="80px" class="drawer-form">
            <el-form-item label="原密码">
              <el-input v-model="pwd.oldPassword" type="password" show-password maxlength="128" placeholder="当前登录密码" />
            </el-form-item>
            <el-form-item label="新密码">
              <el-input v-model="pwd.newPassword" type="password" show-password maxlength="128" placeholder="至少 8 位" />
            </el-form-item>
            <el-form-item label="确认密码">
              <el-input v-model="pwd.confirm" type="password" show-password maxlength="128" @keyup.enter="savePassword" />
            </el-form-item>
            <el-form-item label-width="0" class="form-actions">
              <el-button type="primary" :loading="savingPwd" @click="savePassword">修改密码</el-button>
            </el-form-item>
          </el-form>
          <p class="drawer-tip">修改成功后当前会话立即失效，需要用新密码重新登录。</p>
        </div>
      </el-tab-pane>
    </el-tabs>
  </el-drawer>
</template>

<script setup lang="ts">
// 顶栏「个人信息」抽屉:改本人资料/头像/密码。
// 只作用于登录者自身,入参里没有 id,因此不需要权限码(后端同义,见 internal/router/router.go)。
import { computed, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import type { UploadFile } from 'element-plus'
import { Avatar, Lock, User } from '@element-plus/icons-vue'
import { changePassword, updateProfile, uploadAvatar, type UserProfile } from '../api'
import { useAuthStore } from '../stores/auth'

const open = defineModel<boolean>({ required: true })

const auth = useAuthStore()
const router = useRouter()
const tab = ref('basic')
const user = computed(() => auth.user)

const form = reactive({ nickname: '', email: '', phone: '', signature: '' })
const pwd = reactive({ oldPassword: '', newPassword: '', confirm: '' })
const uploading = ref(false)
const savingProfile = ref(false)
const savingPwd = ref(false)

// 每次打开都从服务端取最新值:store 里的 me 是缓存的,直接编辑可能覆盖别人的改动
watch(open, async (visible) => {
  if (!visible) return
  tab.value = 'basic'
  pwd.oldPassword = ''
  pwd.newPassword = ''
  pwd.confirm = ''
  fill((await auth.fetchMe(true))?.user)
})

function fill(profile?: UserProfile | null) {
  if (!profile) return
  form.nickname = profile.nickname
  form.email = profile.email
  form.phone = profile.phone
  form.signature = profile.signature
}

async function saveProfile() {
  savingProfile.value = true
  try {
    await updateProfile({ ...form })
    fill((await auth.fetchMe(true))?.user)
    ElMessage.success('已更新')
  } catch {
    /* 失败提示由统一拦截器给出 */
  } finally {
    savingProfile.value = false
  }
}

async function onAvatarChange(file: UploadFile) {
  // el-upload 多选会连发 on-change,单飞避免并发上传
  if (!file.raw || uploading.value) return
  uploading.value = true
  try {
    await uploadAvatar(file.raw)
    fill((await auth.fetchMe(true))?.user)
    ElMessage.success('头像已更新')
  } catch {
    /* 失败提示由统一拦截器给出 */
  } finally {
    uploading.value = false
  }
}

async function savePassword() {
  if (pwd.newPassword.length < 8) {
    ElMessage.error('新密码至少 8 位')
    return
  }
  if (pwd.newPassword !== pwd.confirm) {
    ElMessage.error('两次输入的新密码不一致')
    return
  }
  savingPwd.value = true
  try {
    await changePassword(pwd.oldPassword, pwd.newPassword)
    // 服务端已让 access 与 refresh 立即失效:先关抽屉再清凭据,避免用户在失效会话上继续操作
    open.value = false
    auth.invalidate()
    ElMessage.success('密码已修改,请使用新密码重新登录')
    router.replace('/login')
  } catch {
    /* 失败提示由统一拦截器给出 */
  } finally {
    savingPwd.value = false
  }
}
</script>

<style scoped>
.tab-label {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.drawer-body {
  /* 与 el-form 的 label-width 保持一致:头像与提交按钮都对齐到输入框左边缘 */
  --profile-label-width: 80px;
  padding-top: 4px;
}

.avatar-upload,
.form-actions {
  margin-left: var(--profile-label-width);
}

.drawer-form {
  margin-top: 18px;
}

.drawer-tip {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.avatar-circle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 120px;
  height: 120px;
  overflow: hidden;
  border: 1px dashed var(--el-border-color);
  border-radius: 50%;
  background: var(--el-fill-color-lighter);
  cursor: pointer;
  transition: border-color 0.15s;
}

.avatar-circle:hover {
  border-color: var(--el-color-primary);
}

.avatar-circle.is-loading {
  opacity: 0.6;
  pointer-events: none;
}

.avatar-circle__img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-circle__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--el-text-color-regular);
}
</style>
