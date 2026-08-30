import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { ObjectDirective } from 'vue'
import { useAuthStore } from '../stores/auth'
import { perm } from './perm'

const objectPerm = perm as ObjectDirective<HTMLElement, string>

describe('v-perm', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('removes controls for permissions the current user lacks', () => {
    const auth = useAuthStore()
    auth.me = { permissions: ['system:user:list'] } as never
    const parent = document.createElement('div')
    const button = document.createElement('button')
    parent.append(button)

    objectPerm.mounted?.(button, { value: 'system:user:delete' } as never, {} as never, null)

    expect(parent.contains(button)).toBe(false)
  })

  it('keeps controls for granted permissions', () => {
    const auth = useAuthStore()
    auth.me = { permissions: ['system:user:list'] } as never
    const parent = document.createElement('div')
    const button = document.createElement('button')
    parent.append(button)

    objectPerm.mounted?.(button, { value: 'system:user:list' } as never, {} as never, null)

    expect(parent.contains(button)).toBe(true)
  })
})
