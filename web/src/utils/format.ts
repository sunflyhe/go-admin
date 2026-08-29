// 展示层格式化工具

// 时间列统一格式:YYYY-MM-DD HH:mm;withSeconds 时追加秒;空值显示 "-"
export function formatDateTime(value?: string | null, withSeconds = false): string {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  const p = (n: number) => String(n).padStart(2, '0')
  const time = `${p(d.getHours())}:${p(d.getMinutes())}`
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${withSeconds ? `${time}:${p(d.getSeconds())}` : time}`
}

// 文件字节数友好显示:B / KB / MB(保留一位小数)
export function formatSize(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(1)} MB`
}
