<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-logo">
        <el-icon :size="26"><Platform /></el-icon>
      </div>
      <h2 class="login-title">Go Admin</h2>
      <p class="login-sub">Go + Vue3 企业后台开发底座</p>
      <el-form class="login-form" :model="form" @keyup.enter="onLogin">
        <!-- 登录失败用常驻错误条而非 toast:限流/密码错误需要用户停留阅读,3 秒 toast 容易错过 -->
        <el-alert
          v-if="errorMessage"
          class="login-error"
          type="error"
          :title="errorMessage"
          show-icon
          :closable="false"
        />
        <el-form-item>
          <el-input v-model="form.username" placeholder="用户名" size="large" data-test="username">
            <template #prefix>
              <el-icon><User /></el-icon>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item>
          <el-input
            v-model="form.password"
            type="password"
            placeholder="密码"
            size="large"
            show-password
            data-test="password"
          >
            <template #prefix>
              <el-icon><Lock /></el-icon>
            </template>
          </el-input>
        </el-form-item>
        <el-button
          class="login-btn"
          type="primary"
          size="large"
          :loading="loading"
          data-test="submit"
          @click="onLogin"
        >
          登 录
        </el-button>
      </el-form>
    </div>
    <p class="login-copyright">Powered by Go Admin</p>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Lock, Platform, User } from '@element-plus/icons-vue'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const loading = ref(false)
const errorMessage = ref('')
const form = reactive({ username: '', password: '' })

async function onLogin() {
  if (!form.username || !form.password) {
    ElMessage.warning('请输入用户名和密码')
    return
  }
  errorMessage.value = ''
  loading.value = true
  try {
    await auth.login(form.username, form.password)
    ElMessage.success('登录成功')
    router.push((route.query.redirect as string) || '/dashboard')
  } catch (e) {
    // 登录请求已标记 silentError,这里展示常驻错误条(取后端统一文案)
    const ax = e as { response?: { data?: { message?: string } } }
    errorMessage.value = ax.response?.data?.message || '登录失败,请稍后再试'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100vh;
  background:
    radial-gradient(800px 400px at 15% 0%, rgb(37 99 235 / 7%), transparent 60%),
    radial-gradient(700px 380px at 90% 100%, rgb(79 139 255 / 8%), transparent 60%),
    #f8fafc;
}

.login-card {
  width: 420px;
  padding: 44px 40px 36px;
  text-align: center;
  background: #fff;
  border: 1px solid rgb(0 0 0 / 5%);
  border-radius: 24px;
  box-shadow: 0 18px 50px rgb(30 41 59 / 10%);
}

.login-logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border-radius: 18px;
  color: #fff;
  background: linear-gradient(135deg, #4f8bff, #2563eb);
  box-shadow: 0 10px 24px rgb(37 99 235 / 30%);
}

.login-title {
  margin: 18px 0 0;
  font-size: 26px;
  font-weight: 600;
  letter-spacing: 0.5px;
  color: #1e293b;
}

.login-sub {
  margin: 8px 0 0;
  font-size: 13px;
  color: #64748b;
}

.login-error {
  margin-bottom: 18px;
}

.login-form {
  margin-top: 30px;
  text-align: left;
}

.login-form :deep(.el-input__wrapper) {
  border-radius: 8px;
}

.login-btn {
  width: 100%;
  margin-top: 4px;
  border-radius: 8px;
  font-weight: 500;
  letter-spacing: 6px;
  /* 补偿 letter-spacing 造成的视觉偏移 */
  text-indent: 6px;
}

.login-copyright {
  position: absolute;
  bottom: 24px;
  margin: 0;
  font-size: 12px;
  color: #94a3b8;
}
</style>
