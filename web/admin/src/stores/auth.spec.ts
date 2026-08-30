import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const api = vi.hoisted(() => ({
  getMe: vi.fn(),
  login: vi.fn(),
  logout: vi.fn()
}))

vi.mock('../api', () => api)

import { useAuthStore } from './auth'

describe('auth store', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    api.getMe.mockReset()
    api.login.mockReset()
    api.logout.mockReset()
  })

  it('caches /auth/me until explicitly forced', async () => {
    api.getMe.mockResolvedValue({ data: { data: { user: { id: 7 }, permissions: ['system:user:list'], menus: [] } } })
    const auth = useAuthStore()

    await auth.fetchMe()
    await auth.fetchMe()
    await auth.fetchMe(true)

    expect(api.getMe).toHaveBeenCalledTimes(2)
    expect(auth.hasPerm('system:user:list')).toBe(true)
    expect(auth.hasPerm('system:user:delete')).toBe(false)
  })

  it('clears tokens and cached identity after logout', async () => {
    localStorage.setItem('accessToken', 'access')
    localStorage.setItem('refreshToken', 'refresh')
    const auth = useAuthStore()
    auth.token = 'access'
    auth.me = { user: { id: 7 } } as never

    await auth.logout()

    expect(localStorage.getItem('accessToken')).toBeNull()
    expect(localStorage.getItem('refreshToken')).toBeNull()
    expect(auth.token).toBeNull()
    expect(auth.me).toBeNull()
  })
})
