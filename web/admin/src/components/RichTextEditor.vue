<template>
  <div class="rich-editor">
    <Toolbar class="editor-toolbar" :editor="editorRef" :default-config="toolbarConfig" :mode="mode" />
    <Editor
      class="editor-content"
      :style="{ height }"
      v-model="valueHtml"
      :default-config="editorConfig"
      :mode="mode"
      @on-created="handleCreated"
    />
  </div>
</template>

<script setup lang="ts">
// wangEditor 封装:受控 v-model 富文本,图片一律走服务端文章配图接口
// (服务端强制公开可访问,正文 <img> 无法携带 Authorization)。
import '@wangeditor/editor/dist/css/style.css'
import { onBeforeUnmount, ref, shallowRef, watch } from 'vue'
import type { IDomEditor } from '@wangeditor/editor'
import { Editor, Toolbar } from '@wangeditor/editor-for-vue'
import { articleApi } from '../api'

const props = withDefaults(
  defineProps<{
    modelValue: string
    /** 编辑区高度,如 '320px' */
    height?: string
    /** default=完整工具条 / simple=精简工具条 */
    mode?: 'default' | 'simple'
  }>(),
  { height: '320px', mode: 'default' }
)

const emit = defineEmits<{ (e: 'update:modelValue', value: string): void }>()

const editorRef = shallowRef<IDomEditor>()
const valueHtml = ref(props.modelValue)

// 外部 → 编辑器:props 变化时同步(编辑弹窗回填详情);内部 → 外部:交出 HTML
watch(
  () => props.modelValue,
  (value) => {
    if (value !== valueHtml.value) valueHtml.value = value
  }
)
watch(valueHtml, (value) => emit('update:modelValue', value))

const toolbarConfig = { excludeKeys: ['group-video'] }

const editorConfig = {
  placeholder: '开始编写正文…',
  MENU_CONF: {
    uploadImage: {
      // customUpload 接管整个上传流程:上传成功后由 insertFn 把 URL 插进正文
      async customUpload(file: File, insertFn: (url: string, alt: string, href: string) => void) {
        try {
          const res = await articleApi.uploadImage(file)
          insertFn(res.data.data.url, file.name, '')
        } catch {
          // 失败提示由统一请求层负责;这里吞掉 rejection,避免编辑器悬挂
        }
      }
    }
  }
}

function handleCreated(editor: IDomEditor) {
  editorRef.value = editor
}

onBeforeUnmount(() => {
  editorRef.value?.destroy()
})
</script>

<style scoped>
.rich-editor {
  width: 100%;
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  overflow: hidden;
  z-index: 100; /* 压住弹窗内其它元素,保证工具条下拉可见 */
}

.editor-toolbar {
  border-bottom: 1px solid var(--el-border-color);
}
</style>
