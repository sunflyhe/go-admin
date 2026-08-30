# app

面向 C 端用户的 Vue3 SPA(与 `web/admin` 管理端平级)。

## 技术栈与工程约定

与管理端同构:Vue 3 + Vite + TypeScript + Pinia + vue-router + axios(统一请求层 `src/api/request.ts`),ESLint Flat Config + Prettier,Vitest;样式用 Tailwind CSS v4(`@tailwindcss/vite` 插件,入口 `src/styles/tailwind.css`),可与 scoped CSS 混用;Node 版本锁定 24.20.0(最新 LTS,见 `.nvmrc`)。刻意未引入 UI 组件库,确定 C 端视觉方案时再加。

## 命令

```bash
nvm use            # Node 24.20.0
npm install
npm run dev        # 开发模式,端口 5175(避开 admin 的 5173),/api 与 /files 代理到 VITE_PROXY_TARGET
npm run lint && npm run typecheck && npm test && npm run build
```
