# Go Admin 开发约束

本文件面向在本仓库工作的 AI 与开发者。开始修改前先阅读 `README.md`、本文件，以及目标模块已有的代码与测试。若三者冲突，以本文件的架构与安全约束为准，并在交付时说明差异。

## 项目定位

这是可复用的 Go 企业后台开发底座，不是 MineAdmin 功能复刻、低代码平台或多租户 SaaS。优先级固定为：稳定可维护 > 安全边界清楚 > 私有化易部署 > 功能数量。

不要为了“以后可能用到”引入工作流、插件、Redis 强依赖、消息队列、WebSocket、对象存储、数据权限、通用导入或安装向导。确有明确业务需求时，先说明业务价值、边界与维护成本。

## 目录与依赖边界

后端采用 Go 社区常见的 project-layout：

```text
api/
├── cmd/api/             # 启动入口与依赖装配起点
├── internal/
│   ├── handler/         # 唯一直接依赖 Gin 的 HTTP 层
│   ├── service/         # 不依赖 Gin 的业务逻辑
│   ├── model/           # GORM 模型
│   ├── middleware/      # Gin 中间件
│   ├── config/          # 配置加载
│   └── router/          # 路由注册与服务装配
├── pkg/                 # 无业务语义、可跨项目复用的组件
├── migrations/          # SQL Migration
├── configs/             # 配置示例
└── test/                # 测试工具与种子
```

- 不创建另一套 `app/`、`bin/`、`routes/`、`internal/platform/` 目录。
- `internal/` 是项目私有实现；只有确认可脱离本项目复用的无业务组件才能放进 `pkg/`。
- 小型新业务按 `internal/handler/<domain>.go`、`internal/service/<domain>.go`、`internal/model/` 增加代码；当单一业务明显变大时，先说明后再建立按领域分组的子目录。
- 不引入泛化 Repository、DDD、CQRS、Clean Architecture 分层或依赖注入框架，除非用户明确要求。

## Handler 与 Service

### Handler

- Handler 是 HTTP 边界，负责 Gin 路由参数、`ShouldBindJSON` / `ShouldBindQuery`、Header、IP、User-Agent、multipart 文件解析和统一 HTTP 响应。
- HTTP DTO 必须放在 `internal/handler/*_dto.go`，可以保留 Gin 的 `binding` / `form` 标签。
- Handler 必须把 HTTP DTO 显式转换为 Service Input，并使用 `c.Request.Context()` 调用 Service。
- 登录态从 middleware 上下文取得后，转换为 `service.Actor`。登录时的 IP、User-Agent 通过 `service.LoginMeta` 显式传入。

### Service

- `internal/service` 禁止 import `github.com/gin-gonic/gin`。
- Service 仅接收 `context.Context`、业务 Input、`Actor`、`LoginMeta`、文件流等与 HTTP 框架无关的参数。
- Service Input 不得含 Gin `binding` / `form` 标签；Service 不得自行读取 Header、IP、User-Agent、请求体、multipart 或 Gin Context。
- 数据库操作使用 `db.WithContext(ctx)`。
- Service 可被 HTTP、CLI、定时任务或消息入口复用；不要把 HTTP 响应写入 Service。
- `internal/middleware`、`pkg/resp`、`pkg/validate` 允许依赖 Gin，因为它们属于 HTTP 边界，不属于 Service。

## 数据库与接口

- 数据库结构变更只能新增顺序命名的 SQL migration；禁止 GORM `AutoMigrate`。
- 保持 migration 可从空库升级，并考虑已有数据的升级路径；默认不自动执行 down migration。
- 种子数据必须幂等，不能覆盖客户改过的菜单、角色或配置。
- 不随意修改既有 API 路径、响应 JSON、权限码或数据库表结构。确需修改时，先说明兼容性影响与迁移方案。
- 统一响应由 `pkg/resp` 输出；不要在业务代码手写不同格式的 JSON。

## 安全不可回退项

- 每个受保护 API 必须在服务端路由/中间件校验 permission code；前端隐藏按钮不是安全措施。
- 不记录密码、Token、密钥、身份证、手机号、邮箱等未脱敏数据；审计跳过不能由客户端 Header 控制。
- Refresh Token 必须维持轮换、吊销登记和条件更新，避免并发刷新签发多个有效会话。
- 停用用户、重置密码、退出登录、变更角色后，保持已签发凭据及时失效的行为。
- 私有文件必须走鉴权下载；公开文件也必须由服务端确认 `is_public`，不得把整个上传目录静态暴露。
- 上传继续执行大小、扩展名、真实 MIME、路径穿越校验；不得为了方便放宽白名单。
- 超级管理员与内置超级管理员角色保护规则不得移除。
- 数据库密码、JWT 密钥、存储凭证仅存在环境变量或本地配置，绝不提交真实值。

## 前端约束

- Vue 页面通过统一 API 请求层访问后端；不要散落新的未经封装的请求逻辑。
- 菜单与按钮权限以服务端下发的权限为准；切换登录账号时必须重建动态路由。
- 修改渲染行为时，不能只凭 TypeScript 或构建通过声称 UI 已验收；应进行浏览器验证或明确说明未验证。

## 工作方式与验收

1. 修改前：阅读相关代码、测试和 migration，列出预计修改文件；业务规则不明确时停止并询问。
2. 修改中：保持改动聚焦，不重写无关模块，不覆盖用户已有未提交改动。
3. 修改后：至少执行并如实报告：

   ```bash
   cd api && go test ./... && go vet ./... && git diff --check
   cd web && npm run lint && npm run typecheck && npm run build
   ```

4. 涉及 API、认证、权限、上传或前端行为时，补充相应测试。浏览器、真实数据库、Docker 等未运行的验证必须明确标注为未验证，不能由静态检查替代。
5. 交付时说明：修改文件、关键设计选择、测试结果、未验证项与剩余风险。
