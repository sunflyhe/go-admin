// 私有图片缩略图加载器。
// <img src> 带不了 Authorization,私有图片只能经统一请求层取字节再 objectURL;
// 因此必须治理三件事:只加载视口附近的、并发有上限、翻页/离开页面时释放 objectURL。
import { nextTick, onScopeDispose, reactive, ref, watch, type Ref } from 'vue'
import { fileApi, type FileRow } from '../../../api'
import { isImageFile } from './fileDisplay'

const MAX_CONCURRENT = 4
// 缩略图下载的是原图,超过 5MB 不加载,退回类型图标,避免一页卡片吃掉几百 MB
const MAX_THUMB_BYTES = 5 << 20
const ROOT_MARGIN = '200px 0px'

export type ThumbStatus = 'idle' | 'loading' | 'ready' | 'error'

export function useFileThumbnails(rows: Ref<FileRow[]>, enabled?: Ref<boolean>) {
  const rootRef = ref<HTMLElement | null>(null)
  const state = reactive<Record<number, { status: ThumbStatus; url: string }>>({})

  const requested = new Set<number>()
  const queue: number[] = []
  const deferredRelease: number[] = []
  let active = 0
  let previewingId: number | null = null

  const observer = typeof IntersectionObserver !== 'undefined'
    ? new IntersectionObserver(onIntersect, { rootMargin: ROOT_MARGIN })
    : null

  function onIntersect(entries: IntersectionObserverEntry[]) {
    for (const entry of entries) {
      if (!entry.isIntersecting) continue
      const el = entry.target as HTMLElement
      observer?.unobserve(el)
      const id = Number(el.dataset.thumbId)
      if (id) requestThumb(id)
    }
  }

  function stateOf(id: number) {
    if (!state[id]) state[id] = { status: 'idle', url: '' }
    return state[id]
  }

  function requestThumb(id: number) {
    if (requested.has(id)) return
    const row = rows.value.find((r) => r.id === id)
    // 公开图片走 /files 直链,不进队列
    if (!row || !isImageFile(row) || row.isPublic) return
    requested.add(id)
    if (row.size > MAX_THUMB_BYTES) return
    queue.push(id)
    pump()
  }

  function pump() {
    while (active < MAX_CONCURRENT && queue.length) {
      const id = queue.shift() as number
      active++
      void loadOne(id).finally(() => {
        active--
        pump()
      })
    }
  }

  async function loadOne(id: number) {
    stateOf(id).status = 'loading'
    try {
      const { data } = await fileApi.fetchBlob(id)
      // await 期间可能已翻页并被 retain() 回收:此时不能建 objectURL,否则留下无人认领的泄漏
      if (!requested.has(id)) return
      if (!data.type.startsWith('image/')) {
        stateOf(id).status = 'error'
        return
      }
      stateOf(id).url = URL.createObjectURL(data)
      stateOf(id).status = 'ready'
    } catch {
      if (!requested.has(id)) return
      // 文件已被删/磁盘缺失都属常态:静默退回类型图标,卡片仍保留下载入口
      stateOf(id).status = 'error'
    }
  }

  function release(id: number) {
    if (state[id]?.url) URL.revokeObjectURL(state[id].url)
    delete state[id]
    requested.delete(id)
  }

  // 只保留当前页需要的缩略图;预览中的那张延后到关闭时释放,避免大图突然黑掉
  function retain(keepIds: number[]) {
    const keep = new Set(keepIds)
    for (const key of Object.keys(state)) {
      const id = Number(key)
      if (keep.has(id)) continue
      if (id === previewingId) {
        deferredRelease.push(id)
        continue
      }
      release(id)
    }
    for (let i = queue.length - 1; i >= 0; i--) {
      if (!keep.has(queue[i])) {
        const [dropped] = queue.splice(i, 1)
        requested.delete(dropped)
      }
    }
  }

  function setPreviewing(id: number | null) {
    previewingId = id
    if (id === null) {
      const pending = deferredRelease.splice(0)
      for (const pid of pending) release(pid)
    }
  }

  function thumbTargets(): HTMLElement[] {
    return rootRef.value
      ? Array.from(rootRef.value.querySelectorAll<HTMLElement>('[data-thumb-id]'))
      : []
  }

  function observeAll() {
    const targets = thumbTargets()
    if (!observer) {
      // 环境不支持 IntersectionObserver 时退化为全量加载
      for (const el of targets) {
        const id = Number(el.dataset.thumbId)
        if (id) requestThumb(id)
      }
      return
    }
    observer.disconnect()
    for (const el of targets) observer.observe(el)
  }

  watch(
    [rows, () => enabled?.value ?? true],
    async ([list, on]) => {
      // 列表模式不渲染图块:必须显式回收,否则一页 blob 会一直挂在内存里
      if (!on) {
        retain([])
        observer?.disconnect()
        return
      }
      retain(list.filter(isImageFile).map((r) => r.id))
      await nextTick()
      observeAll()
    },
    { flush: 'post' }
  )

  onScopeDispose(() => {
    observer?.disconnect()
    for (const key of Object.keys(state)) release(Number(key))
    queue.length = 0
    requested.clear()
  })

  return { rootRef, state, setPreviewing }
}
