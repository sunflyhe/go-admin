<template>
  <PageHeader :description="isEdit ? `编辑文章:${form.title || '加载中…'}` : '撰写新文章,保存草稿或直接发布。'">
    <template #extra>
      <el-button @click="goBack">返回列表</el-button>
    </template>
  </PageHeader>
  <el-card v-loading="loading">
    <el-form :model="form" label-width="70px" class="article-form">
      <el-form-item label="标题" required>
        <el-input v-model="form.title" maxlength="128" show-word-limit placeholder="请输入文章标题" />
      </el-form-item>
      <el-form-item label="分类">
        <el-select v-model="form.categoryId" style="width: 240px">
          <el-option label="未分类" :value="0" />
          <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="摘要">
        <el-input v-model="form.summary" type="textarea" :rows="2" maxlength="255" show-word-limit placeholder="列表展示用,可留空" />
      </el-form-item>
      <el-form-item label="状态">
        <el-radio-group v-model="form.status">
          <el-radio :value="1">草稿</el-radio>
          <el-radio :value="2">发布</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="正文" required>
        <RichTextEditor v-model="form.content" height="480px" />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
        <el-button @click="goBack">取消</el-button>
      </el-form-item>
    </el-form>
  </el-card>
</template>

<script setup lang="ts">
// 文章编辑页(独立路由而非弹窗):/article/article/edit?id=xx,不带 id 为新建。
// 权限由入口按钮(v-perm)与后端接口双重控制,直连 URL 会在保存时被服务端 403。
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { articleApi, articleCategoryApi, type ArticleCategory } from '../../../api'
import PageHeader from '../../../components/PageHeader.vue'
import RichTextEditor from '../../../components/RichTextEditor.vue'

const route = useRoute()
const router = useRouter()

const categories = ref<ArticleCategory[]>([])
const loading = ref(false)
const saving = ref(false)
const form = reactive({ id: 0, categoryId: 0, title: '', summary: '', content: '', status: 1 })

const isEdit = computed(() => form.id > 0)

onMounted(async () => {
  // 分类下拉加载失败时降级为"未分类",不阻塞编辑(需 article:category:list)
  try {
    const { data } = await articleCategoryApi.list()
    categories.value = data.data
  } catch {
    categories.value = []
  }
  const id = Number(route.query.id)
  if (id > 0) {
    loading.value = true
    try {
      const { data } = await articleApi.get(id)
      const detail = data.data
      Object.assign(form, {
        id: detail.id,
        categoryId: detail.categoryId,
        title: detail.title,
        summary: detail.summary,
        content: detail.content,
        status: detail.status
      })
    } finally {
      loading.value = false
    }
  }
})

// 空正文兜底:编辑器默认产出 <p><br></p> 这类"看起来有内容"的空段落
function isContentEmpty(html: string) {
  return html.replace(/<[^>]*>/g, '').trim() === ''
}

async function save() {
  if (!form.title.trim()) {
    ElMessage.warning('请输入标题')
    return
  }
  if (isContentEmpty(form.content)) {
    ElMessage.warning('请填写正文')
    return
  }
  saving.value = true
  try {
    const payload = {
      categoryId: form.categoryId,
      title: form.title.trim(),
      summary: form.summary,
      content: form.content,
      status: form.status
    }
    if (form.id) {
      await articleApi.update(form.id, payload)
    } else {
      await articleApi.create(payload)
    }
    ElMessage.success('保存成功')
    goBack()
  } finally {
    saving.value = false
  }
}

function goBack() {
  router.push('/article/article')
}
</script>

<style scoped>
.article-form {
  max-width: 960px;
}
</style>
