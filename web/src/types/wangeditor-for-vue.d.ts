// @wangeditor/editor-for-vue@5 的类型文件存在于 dist/src/index.d.ts,
// 但其 package.json 的 exports 未声明 types,vue-tsc 解析不到,这里手动补模块声明。
// 声明保持宽松:具体配置项以 @wangeditor/editor 自身的类型为准。
declare module '@wangeditor/editor-for-vue' {
  import type { DefineComponent } from 'vue'

  type EditorProps = {
    /** 编辑器 HTML(v-model) */
    modelValue?: string
    /** 编辑器配置:placeholder、MENU_CONF 等 */
    defaultConfig?: Record<string, unknown>
    defaultContent?: unknown[]
    defaultHtml?: string
    /** default=完整工具条 / simple=精简工具条 */
    mode?: string
    style?: Record<string, string | number>
  }

  type ToolbarProps = {
    editor?: unknown
    defaultConfig?: Record<string, unknown>
    mode?: string
  }

  type Emits = {
    (e: 'update:modelValue', value: string): void
    (e: 'onCreated', editor: unknown): void
    (e: 'onChange', editor: unknown): void
    (e: 'onDestroyed', editor: unknown): void
  }

  export const Editor: DefineComponent<EditorProps, Emits>
  export const Toolbar: DefineComponent<ToolbarProps>
}
