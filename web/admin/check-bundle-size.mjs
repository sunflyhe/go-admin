// 构建产物体积预算检查:vite build 之后由 `npm run check:bundle` 执行。
// 超出任一预算(或找不到对应产物)即退出码 1,使 build 失败,防止依赖膨胀悄然发生。
// 调整预算前先确认必要性:入口 JS 影响首屏加载,富文本单独放宽的前提是它始终待在懒加载 chunk 里。
import { readFile, readdir, stat } from 'node:fs/promises'
import { resolve } from 'node:path'

const distDir = resolve('dist')
const budgets = [
  { label: '入口 JavaScript', pattern: /<script[^>]+src="[^"]*\/(assets\/[^"]+\.js)"/, limit: 300 * 1024 },
  { label: '最大 CSS', pattern: /assets\/(.+\.css)$/u, limit: 64 * 1024, largest: true },
  { label: '富文本懒加载包', pattern: /assets\/(edit-[^/]+\.js)$/u, limit: 900 * 1024 }
]

const indexHtml = await readFile(resolve(distDir, 'index.html'), 'utf8')
const files = await readdir(resolve(distDir, 'assets'))
let failed = false

for (const budget of budgets) {
  let candidates
  if (budget.largest) {
    candidates = files.filter((file) => budget.pattern.test(`assets/${file}`))
  } else if (budget.label === '入口 JavaScript') {
    const match = indexHtml.match(budget.pattern)
    candidates = match ? [match[1].replace('assets/', '')] : []
  } else {
    candidates = files.filter((file) => budget.pattern.test(`assets/${file}`))
  }
  if (candidates.length === 0) {
    console.error(`未找到${budget.label}产物`)
    failed = true
    continue
  }
  const sizes = await Promise.all(
    candidates.map(async (file) => ({ file, size: (await stat(resolve(distDir, 'assets', file))).size }))
  )
  const result = budget.largest ? sizes.reduce((max, item) => (item.size > max.size ? item : max)) : sizes[0]
  const status = result.size <= budget.limit ? 'OK' : '超出'
  console.log(
    `${status} ${budget.label}: ${(result.size / 1024).toFixed(1)} KiB / ${(budget.limit / 1024).toFixed(0)} KiB (${result.file})`
  )
  failed ||= result.size > budget.limit
}

if (failed) process.exit(1)
