// 菜单图标白名单：数据库图标名映射为实际组件，避免为动态 <component> 全量注册图标包。
import {
  Avatar,
  Collection,
  Document,
  Files,
  Key,
  Menu,
  Notebook,
  Odometer,
  Operation,
  Setting,
  SetUp,
  Tickets,
  User
} from '@element-plus/icons-vue'
import type { Component } from 'vue'

const menuIcons: Record<string, Component> = {
  Avatar,
  Collection,
  Document,
  Files,
  Key,
  Menu,
  Notebook,
  Odometer,
  Operation,
  Setting,
  SetUp,
  Tickets,
  User
}

// 客户自定义了未收录的图标名时保留通用菜单图标，避免菜单项因按需加载而突然失去图标。
export function menuIcon(name: string): Component {
  return menuIcons[name] ?? Menu
}
