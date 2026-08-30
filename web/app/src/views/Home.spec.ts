import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import Home from './Home.vue'

describe('Home', () => {
  it('渲染品牌标题', () => {
    const wrapper = mount(Home, { global: { plugins: [createPinia()] } })
    expect(wrapper.find('h1').text()).toContain('Go Admin')
  })
})
