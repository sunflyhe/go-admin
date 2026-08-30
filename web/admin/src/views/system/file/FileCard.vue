<template>
  <li
    class="file-card"
    :class="{ 'is-previewable': previewable, 'is-selected': selected }"
    :title="hint"
    @click="previewable && emit('preview', file)"
  >
    <div class="file-card__media">
      <el-checkbox
        class="file-card__check"
        :model-value="selected"
        :aria-label="`选择 ${file.originName}`"
        @click.stop
        @change="emit('select', file)"
      />
      <img v-if="thumbUrl" :src="thumbUrl" :alt="file.originName" class="file-card__img" decoding="async" />
      <el-icon v-else class="file-card__icon" :size="40"><component :is="icon" /></el-icon>

      <div class="file-card__actions">
        <el-button circle :icon="Download" title="下载" @click.stop="emit('download', file)" />
        <el-button v-perm="'system:file:delete'" circle :icon="Delete" title="删除" @click.stop="emit('remove', file)" />
      </div>
    </div>
    <!-- 文件名移到图块下方:压在缩略图上的渐变标题在浅底图片上根本读不清 -->
    <p class="file-card__name">{{ file.originName }}</p>
    <p class="file-card__meta">{{ formatSize(file.size) }} · {{ formatDateTime(file.createdAt) }}</p>
  </li>
</template>

<script setup lang="ts">
// 单张文件卡片:纯展示 + 选择/下载/删除/预览四个事件,数据与缩略图状态由页面注入。
import { computed } from 'vue'
import { Delete, Download } from '@element-plus/icons-vue'
import type { FileRow } from '../../../api'
import { formatDateTime, formatSize } from '../../../utils/format'
import { fileIcon, isImageFile } from './fileDisplay'

const props = defineProps<{ file: FileRow; thumbUrl: string; selected: boolean }>()

const emit = defineEmits<{
  download: [file: FileRow]
  remove: [file: FileRow]
  preview: [file: FileRow]
  select: [file: FileRow]
}>()

const icon = computed(() => fileIcon(props.file))
const previewable = computed(() => isImageFile(props.file))
const hint = computed(
  () => `${props.file.originName}｜${props.file.uploader || '未知'} 上传 · ${props.file.isPublic ? '公开' : '私有'}`
)
</script>

<style scoped>
.file-card {
  list-style: none;
  cursor: default;
}

.file-card.is-previewable {
  cursor: zoom-in;
}

.file-card__media {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  aspect-ratio: 1 / 1;
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  transition: border-color 0.15s;
}

/* 选中态用边框+浅底表达,不遮挡缩略图本身 */
.file-card.is-selected .file-card__media {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.file-card__img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.file-card__check {
  position: absolute;
  top: 4px;
  left: 6px;
  z-index: 2;
  height: auto;
  padding: 2px;
  border-radius: 4px;
  background: rgb(255 255 255 / 86%);
  opacity: 0;
  transition: opacity 0.15s;
}

.file-card__media:hover .file-card__check,
.file-card.is-selected .file-card__check {
  opacity: 1;
}

.file-card__actions {
  position: absolute;
  top: 6px;
  right: 6px;
  display: flex;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.15s;
}

.file-card__actions :deep(.el-button) {
  background: rgb(255 255 255 / 86%);
  border: none;
}

.file-card__media:hover .file-card__actions {
  opacity: 1;
}

.file-card__name {
  margin: 8px 0 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-size: 13px;
  color: var(--el-text-color-primary);
}

.file-card__meta {
  margin: 2px 0 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-size: 11px;
  color: var(--el-text-color-secondary);
}
</style>
