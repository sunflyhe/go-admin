// 登录态与权限存储。
import { defineStore } from 'pinia'
import { getMe, login as apiLogin, logout as apiLogout, type MeResp, type UserProfile } from '../api'

interface AuthState {
  me: MeResp | null
  loaded: boolean
  token: string | null
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    me: null,
    loaded: false,
    token: localStorage.getItem('accessToken')
  }),
  getters: {
    isLoggedIn: (state) => !!state.token,
    user(): UserProfile | null {
      return this.me?.user ?? null
    },
    permissions(): string[] {
      return this.me?.permissions ?? []
    }
  },
  actions: {
    async login(username: string, password: string) {
      const { data } = await apiLogin(username, password)
      localStorage.setItem('accessToken', data.data.accessToken)
      localStorage.setItem('refreshToken', data.data.refreshToken)
      this.token = data.data.accessToken
    },
    async fetchMe(force = false) {
      if (this.loaded && !force) return this.me
      const { data } = await getMe()
      this.me = data.data
      this.loaded = true
      return this.me
    },
    async logout() {
      try {
        await apiLogout()
      } finally {
        localStorage.removeItem('accessToken')
        localStorage.removeItem('refreshToken')
        this.me = null
        this.loaded = false
        this.token = null
      }
    },
    invalidate() {
      localStorage.removeItem('accessToken')
      localStorage.removeItem('refreshToken')
      this.me = null
      this.loaded = false
      this.token = null
    },
    hasPerm(code: string): boolean {
      if (!this.me) return false
      const perms = this.me.permissions
      if (perms.includes('*')) return true
      return perms.includes(code)
    }
  }
})
