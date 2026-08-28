// v-perm 按钮级权限指令:无权限时移除元素。仅为体验优化,真正的安全边界在服务端。
import type { Directive } from 'vue'
import { useAuthStore } from '../stores/auth'

export const perm: Directive<HTMLElement, string> = {
  mounted(el, binding) {
    const auth = useAuthStore()
    if (!auth.hasPerm(binding.value)) {
      el.parentNode?.removeChild(el)
    }
  }
}
