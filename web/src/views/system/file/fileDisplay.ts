// 文件展示层的极小纯函数:是否图片、类型图标名、公开直链。
import type { FileRow } from '../../../api'

// mime 是服务端按文件内容嗅探出来的权威值,前端不再维护一份扩展名清单。
export function isImageFile(row: FileRow): boolean {
  return row.mime.startsWith('image/')
}

// 与后端"视频"标签同源(按 mime 前缀)。当前上传白名单不含音视频,
// 该分支只在服务端放开白名单后生效,前端不因此放宽任何校验。
export function isVideoFile(row: FileRow): boolean {
  return row.mime.startsWith('video/')
}

// 公开文件由服务端 inline 输出,可以直接给 <img src>,不必再走鉴权下载。
export function publicSrc(row: FileRow): string {
  return `/files/${row.storePath}`
}

// 图标名对应 main.ts 全局注册的 element-plus 图标组件。
// .zip 必须先看扩展名:.docx/.xlsx 是 ZIP 容器,嗅探结果同样是 application/zip。
export function fileIcon(row: FileRow): string {
  if (row.ext === '.zip') return 'Box'
  if (isImageFile(row)) return 'Picture'
  if (isVideoFile(row)) return 'VideoCamera'
  if (row.mime === 'application/pdf') return 'Document'
  if (row.mime.startsWith('text/')) return 'Memo'
  return 'Files'
}
