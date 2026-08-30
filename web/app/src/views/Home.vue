<script setup lang="ts">
// 项目介绍首页:纯静态展示,无接口依赖。
const features = [
  {
    title: 'RBAC 权限',
    desc: '菜单、按钮级权限码全部在服务端强制校验,前端隐藏只是体验,越权请求一律 403。',
    icon: 'shield'
  },
  {
    title: '双令牌认证',
    desc: 'access + refresh 双 token,刷新轮换与吊销登记,退出/停用/改密即时全端失效。',
    icon: 'key'
  },
  {
    title: '审计与日志',
    desc: '操作日志、登录日志开箱即用,密码/Token/手机号等敏感字段自动脱敏落库。',
    icon: 'list'
  },
  {
    title: '文件中心',
    desc: '扩展名 + 真实 MIME 双白名单上传,公开/私有文件分策略鉴权下载,拒绝路径穿越。',
    icon: 'folder'
  },
  {
    title: '内容资讯',
    desc: '文章分类与富文本内容管理,配图独立上传权限,已发布内容可直接对外输出。',
    icon: 'doc'
  },
  {
    title: '参数与字典',
    desc: '系统参数与字典类型统一维护,业务侧按键读取,改动即时生效无需发版。',
    icon: 'sliders'
  }
]

const stack = ['Go 1.27', 'Gin', 'GORM', 'MySQL', 'Vue 3', 'TypeScript', 'Vite', 'Pinia', 'Element Plus']

const backendTree = [
  'api/',
  '├── cmd/api            启动入口与依赖装配',
  '├── internal/',
  '│   ├── handler        HTTP 边界(DTO 绑定与转换)',
  '│   ├── service        业务逻辑(不依赖 Gin)',
  '│   ├── middleware     JWT 鉴权 / 权限码 / 审计',
  '│   └── router         按端拆分:admin / api',
  '├── pkg                可复用组件(JWT/日志/响应…)',
  '└── migrations         内嵌 SQL 迁移,启动自动执行'
]

const frontendTree = [
  'web/',
  '├── admin              管理端 SPA(Vue3 + Element Plus)',
  '│   ├── views          页面 + 动态路由(服务端菜单)',
  '│   ├── api            统一请求层(401 自愈刷新)',
  '│   └── directives     v-perm 按钮级权限',
  '└── app                C 端 SPA(本页面)',
]

const securities = [
  { k: '密码只存 bcrypt', v: '登录失败统一文案,用户名+IP 双维度限流锁定' },
  { k: '凭据可即时作废', v: 'token_version 机制,异常会话一次变更全部失效' },
  { k: '超管保护', v: '内置账号与内置角色不可删改,系统永远有管理入口' },
  { k: '私有化交付', v: '一个静态二进制 + 一份配置即完整交付,内网无外部依赖' }
]

function iconPath(name: string): string {
  const map: Record<string, string> = {
    shield: 'M12 2l8 4v6c0 5-3.5 8.5-8 10-4.5-1.5-8-5-8-10V6l8-4z',
    key: 'M14 2a6 6 0 0 0-5.8 7.6L2 15.8V20h4.2l1-1v-2h2v-2h2l1.2-1.2A6 6 0 1 0 14 2zm2 3.5a1.5 1.5 0 1 1 0 3 1.5 1.5 0 0 1 0-3z',
    list: 'M4 5h3v3H4V5zm5 0h11v2H9V5zM4 10.5h3v3H4v-3zm5 0h11v2H9v-2zM4 16h3v3H4v-3zm5 0h11v2H9v-2z',
    folder: 'M3 5h6l2 2h10v12H3V5zm9 5v3H9v2h3v3h2v-3h3v-2h-3v-3h-2z',
    doc: 'M6 2h9l5 5v15H6V2zm8 1.5V8h4.5L14 3.5zM8 11h8v2H8v-2zm0 4h8v2H8v-2z',
    sliders: 'M4 5h9v2H4V5zm13 0h3v2h-3V5zM4 11h3v2H4v-2zm7 0h9v2h-9v-2zM4 17h9v2H4v-2zm13 0h3v2h-3v-2z'
  }
  return map[name] ?? map.shield
}
</script>

<template>
  <div class="landing">
    <header class="nav">
      <div class="nav-inner">
        <span class="brand">
          <span class="brand-mark">G</span>
          Go Admin
        </span>
        <nav class="nav-links">
          <a href="#features">特性</a>
          <a href="#architecture">架构</a>
          <a href="#security">安全</a>
        </nav>
      </div>
    </header>

    <section class="hero">
      <div class="hero-glow" aria-hidden="true"></div>
      <p class="hero-badge">开箱即用的企业后台开发底座 · v1.0</p>
      <h1>
        <span class="hero-brand">Go Admin</span>
        <span class="hero-sub">把权限、审计与交付变成默认能力</span>
      </h1>
      <p class="hero-desc">
        Go + Vue3 前后端分离的管理中台骨架:RBAC 权限、双令牌认证、审计日志、文件中心、内容资讯开箱即用,
        私有化一键交付,让你直接从业务写起。
      </p>
      <div class="hero-actions">
        <a class="btn btn-primary" href="#features">了解特性</a>
        <a class="btn btn-ghost" href="#architecture">查看架构</a>
      </div>
      <ul class="hero-stack">
        <li v-for="s in stack" :key="s">{{ s }}</li>
      </ul>
    </section>

    <section id="features" class="section">
      <h2 class="section-title">核心特性</h2>
      <p class="section-desc">每一项都是生产环境反复打磨过的默认能力,不是示例代码。</p>
      <div class="feature-grid">
        <article v-for="f in features" :key="f.title" class="feature-card">
          <span class="feature-icon">
            <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor" aria-hidden="true">
              <path :d="iconPath(f.icon)" />
            </svg>
          </span>
          <h3>{{ f.title }}</h3>
          <p>{{ f.desc }}</p>
        </article>
      </div>
    </section>

    <section id="architecture" class="section section-alt">
      <h2 class="section-title">工程架构</h2>
      <p class="section-desc">project-layout 分层,Handler→Service 边界清晰,Service 不依赖任何 Web 框架。</p>
      <div class="arch-grid">
        <div class="arch-card">
          <h3>后端 api/</h3>
          <pre class="arch-tree">{{ backendTree.join('\n') }}</pre>
        </div>
        <div class="arch-card">
          <h3>前端 web/</h3>
          <pre class="arch-tree">{{ frontendTree.join('\n') }}</pre>
        </div>
      </div>
      <p class="arch-note">数据库变更全部走顺序 SQL 迁移(内嵌进二进制,启动自动执行),禁止 AutoMigrate;CI 覆盖 Go 测试、前端三件套与发布包构建。</p>
    </section>

    <section id="security" class="section section-dark">
      <h2 class="section-title">安全不是开关,而是底线</h2>
      <p class="section-desc">以下保护内建于代码,不因部署环境而妥协。</p>
      <div class="security-grid">
        <div v-for="s in securities" :key="s.k" class="security-item">
          <strong>{{ s.k }}</strong>
          <span>{{ s.v }}</span>
        </div>
      </div>
    </section>

    <!-- 页脚:Tailwind 工具类写法试点,其余分段仍为 scoped CSS,两种方式可混用 -->
    <footer class="flex flex-col items-center gap-1.5 bg-[#f7f8fc] px-5 pt-8 pb-10 text-[13px] text-[#4b5060]">
      <span>Go Admin · 企业后台开发底座</span>
      <span class="text-xs text-[#9aa0ae]">Go + Vue3 · 私有化友好 · 单仓交付</span>
    </footer>
  </div>
</template>

<style scoped>
.landing {
  min-height: 100vh;
  background: #f7f8fc;
  color: #1e2330;
}

/* ---- 导航 ---- */
.nav {
  position: fixed;
  top: 0;
  right: 0;
  left: 0;
  z-index: 10;
  background: rgba(13, 16, 32, 0.72);
  backdrop-filter: blur(14px);
  -webkit-backdrop-filter: blur(14px);
}

.nav-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  max-width: 1080px;
  margin: 0 auto;
  padding: 14px 20px;
}

.brand {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  color: #fff;
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.brand-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 8px;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  font-size: 13px;
}

.nav-links {
  display: flex;
  gap: 22px;
}

.nav-links a {
  color: rgba(255, 255, 255, 0.72);
  font-size: 13px;
  text-decoration: none;
  transition: color 0.15s;
}

.nav-links a:hover {
  color: #fff;
}

/* ---- Hero ---- */
.hero {
  position: relative;
  overflow: hidden;
  padding: 150px 20px 90px;
  background: #0d1020;
  color: #fff;
  text-align: center;
}

.hero-glow {
  position: absolute;
  top: -180px;
  left: 50%;
  width: 720px;
  height: 480px;
  background:
    radial-gradient(400px 260px at 40% 45%, rgba(99, 102, 241, 0.5), transparent 70%),
    radial-gradient(380px 240px at 62% 55%, rgba(139, 92, 246, 0.42), transparent 70%),
    radial-gradient(300px 220px at 50% 70%, rgba(56, 189, 248, 0.25), transparent 70%);
  filter: blur(8px);
  transform: translateX(-50%);
  pointer-events: none;
}

.hero-badge {
  position: relative;
  display: inline-block;
  margin: 0 0 22px;
  padding: 6px 14px;
  border: 1px solid rgba(255, 255, 255, 0.18);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.06);
  font-size: 12.5px;
  color: rgba(255, 255, 255, 0.85);
}

.hero h1 {
  position: relative;
  margin: 0;
}

.hero-brand {
  display: block;
  font-size: clamp(44px, 8vw, 72px);
  font-weight: 700;
  letter-spacing: 0.01em;
  background: linear-gradient(100deg, #ffffff 30%, #b7bbff 70%, #8f9bff);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
}

.hero-sub {
  display: block;
  margin-top: 14px;
  font-size: clamp(17px, 3vw, 22px);
  font-weight: 500;
  color: rgba(255, 255, 255, 0.9);
}

.hero-desc {
  position: relative;
  max-width: 620px;
  margin: 20px auto 0;
  font-size: 15px;
  line-height: 1.8;
  color: rgba(255, 255, 255, 0.62);
}

.hero-actions {
  position: relative;
  display: flex;
  justify-content: center;
  gap: 14px;
  margin-top: 34px;
}

.btn {
  display: inline-block;
  padding: 11px 26px;
  border-radius: 12px;
  font-size: 14px;
  text-decoration: none;
  transition:
    transform 0.15s,
    box-shadow 0.15s,
    background 0.15s;
}

.btn-primary {
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  color: #fff;
  box-shadow: 0 8px 24px rgba(99, 102, 241, 0.4);
}

.btn-primary:hover {
  transform: translateY(-2px);
  box-shadow: 0 12px 30px rgba(99, 102, 241, 0.5);
}

.btn-ghost {
  border: 1px solid rgba(255, 255, 255, 0.22);
  color: rgba(255, 255, 255, 0.88);
}

.btn-ghost:hover {
  background: rgba(255, 255, 255, 0.08);
}

.hero-stack {
  position: relative;
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
  max-width: 640px;
  margin: 38px auto 0;
  padding: 0;
  list-style: none;
}

.hero-stack li {
  padding: 5px 12px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.05);
  font-size: 12px;
  color: rgba(255, 255, 255, 0.66);
  font-variant-numeric: tabular-nums;
}

/* ---- 通用段落 ---- */
.section {
  max-width: 1080px;
  margin: 0 auto;
  padding: 76px 20px;
  text-align: center;
}

.section-alt {
  max-width: none;
  background: #eef0f8;
}

.section-alt > * {
  max-width: 1080px;
  margin-left: auto;
  margin-right: auto;
}

.section-title {
  margin: 0;
  font-size: 26px;
  letter-spacing: 0.01em;
}

.section-desc {
  margin: 10px 0 0;
  font-size: 14px;
  color: #6b7280;
}

/* ---- 特性 ---- */
.feature-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 16px;
  margin-top: 40px;
  text-align: left;
}

.feature-card {
  padding: 24px;
  border: 1px solid #e7e9f2;
  border-radius: 18px;
  background: #fff;
  transition:
    transform 0.18s,
    box-shadow 0.18s;
}

.feature-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 14px 34px rgba(30, 35, 48, 0.1);
}

.feature-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 12px;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.14), rgba(139, 92, 246, 0.16));
  color: #6366f1;
}

.feature-card h3 {
  margin: 14px 0 8px;
  font-size: 15.5px;
}

.feature-card p {
  margin: 0;
  font-size: 13.5px;
  line-height: 1.75;
  color: #6b7280;
}

/* ---- 架构 ---- */
.arch-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 16px;
  margin-top: 40px;
}

.arch-card {
  padding: 22px 24px;
  border-radius: 18px;
  background: #fff;
  border: 1px solid #e3e6f0;
  text-align: left;
}

.arch-card h3 {
  margin: 0 0 12px;
  font-size: 14px;
  color: #4b5060;
}

.arch-tree {
  margin: 0;
  padding: 16px;
  border-radius: 12px;
  background: #0d1020;
  color: #a5f3d0;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12.5px;
  line-height: 1.75;
  overflow-x: auto;
}

.arch-note {
  margin: 22px auto 0;
  max-width: 820px;
  font-size: 13px;
  line-height: 1.8;
  color: #6b7280;
}

/* ---- 安全 ---- */
.section-dark {
  max-width: none;
  background: #0d1020;
  color: #fff;
}

.section-dark > * {
  max-width: 1080px;
  margin-left: auto;
  margin-right: auto;
}

.section-dark .section-desc {
  color: rgba(255, 255, 255, 0.55);
}

.security-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(230px, 1fr));
  gap: 14px;
  margin-top: 40px;
}

.security-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 20px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.04);
  text-align: left;
}

.security-item strong {
  font-size: 14px;
  background: linear-gradient(100deg, #c7cbff, #9ba4ff);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
}

.security-item span {
  font-size: 12.5px;
  line-height: 1.7;
  color: rgba(255, 255, 255, 0.58);
}

/* ---- 页脚样式已迁移为 Tailwind 工具类(见模板) ---- */
</style>
