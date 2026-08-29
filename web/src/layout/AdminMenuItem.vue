<template>
  <!-- 目录:可继续嵌套(自身递归);页面:叶子节点 -->
  <el-sub-menu v-if="node.type === 1 && node.children?.length" :index="node.path">
    <template #title>
      <el-icon v-if="node.icon"><component :is="menuIcon(node.icon)" /></el-icon>
      <span>{{ node.name }}</span>
    </template>
    <AdminMenuItem v-for="child in node.children" :key="child.id" :node="child" />
  </el-sub-menu>
  <el-menu-item v-else-if="node.type === 2" :index="node.path">
    <el-icon v-if="node.icon"><component :is="menuIcon(node.icon)" /></el-icon>
    <span>{{ node.name }}</span>
  </el-menu-item>
</template>

<script lang="ts">
import { defineComponent } from 'vue'
import type { MenuNode } from '../api'
import { menuIcon } from './menuIcons'

// 递归菜单节点:目录(type=1)嵌套渲染,页面(type=2)作为叶子。
// 通过 name 选项实现模板内自引用。
export default defineComponent({
  name: 'AdminMenuItem',
  methods: { menuIcon },
  props: {
    node: { type: Object as () => MenuNode, required: true }
  }
})
</script>
