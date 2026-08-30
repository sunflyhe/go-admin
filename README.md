# Go Admin — Go 企业后台开发底座

可复用的企业后台开发底座:登录认证、账号/角色/菜单 RBAC、操作审计、登录日志、本地文件上传、Excel 导出。
定位是**新业务模块的起点**,不是低代码平台,也不是 SaaS。优先级:稳定可维护 > 安全边界清楚 > 私有化易部署 > 功能数量。

技术栈:Go + Gin + GORM(仅查询映射)+ MySQL 8(golang-migrate 管理 schema)+ JWT(bcrypt 密码)+ Vue 3 + TypeScript + Element Plus + Pinia。

## 目录结构

后端采用 Go 社区通行的分层布局(golang-standards/project-layout 风格),按技术角色横切分层:

```text
api/                     Go 后端 API(单二进制)
├── cmd/api/main.go      启动入口
├── internal/            私有代码(编译器禁止外部模块导入)
│   ├── handler/         HTTP 处理器(参数绑定+响应)
│   ├── service/         不依赖 Gin 的业务逻辑(业务 Input/Output、Actor 等)
│   ├── model/           GORM 数据模型
│   ├── middleware/      请求 ID/访问日志/恢复/认证/权限校验
│   ├── config/          配置加载(文件 + 环境变量覆盖)
│   └── router/          路由注册与服务装配
├── pkg/                 无业务语义、可复用的通用包(JWT、DB、日志、迁移、错误码、响应、分页、校验)
├── migrations/          SQL migration(embed,启动时自动执行 up)
├── test/                测试基础设施(内存库 + 种子)
└── configs/             配置示例(configs/config.example.yaml)
web/
├── admin/    Vue3 管理端
└── app/      C 端用户前端(预留,尚未初始化)
deploy/    部署套件:发布包构建脚本、systemd/nginx 模板、部署手册(deploy/binary/)
```

约定:HTTP DTO 放在 `internal/handler/*_dto.go`，Handler 显式转换后以 `context.Context` 调用不依赖 Gin 的 Service；新增业务模块时,handler/service/各自加文件、model 进 model 包,路由在 internal/router 注册;`internal/` 与 `pkg/` 的分界是"是否可被其他项目复用"。

## 快速开始(本地)

### 1. 准备 MySQL

```bash
mysql -uroot -p -e "CREATE DATABASE go_admin DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
```

### 2. 配置

```bash
cd api
cp configs/config.example.yaml configs/config.yaml
# 编辑 configs/config.yaml:填写 mysql.dsn 与 jwt.secret(至少 16 位,建议 32 位随机串)
```

或完全用环境变量(优先级高于文件):`ADMIN_MYSQL_DSN`、`ADMIN_JWT_SECRET`、`ADMIN_SERVER_ADDR` 等,见 `configs/config.example.yaml` 注释。

### 3. 启动(空库自动迁移 + 种子)

```bash
cd api
go run ./cmd/api -config configs/config.yaml
```

服务启动时会自动执行未应用的迁移与幂等种子数据;也可用 `api/migrations/` 下的 SQL 手工执行。

开发时推荐用 [air](https://github.com/air-verse/air) 热重载代替 `go run`(改动 `.go`/`.sql`/`.yaml` 自动重编译重启,约 1 秒):

```bash
go install github.com/air-verse/air@latest
cd api && air     # 配置见 api/.air.toml,纯开发工具,不进部署链
```

### 4. 前端

```bash
cd web/admin
nvm use                 # 使用仓库锁定的 Node 24.20.0
npm install
npm run dev        # 开发模式,访问 http://localhost:5173/admin/,/admin-api 代理到 localhost:8080
npm run build      # 生产构建,产物在 web/admin/dist(路由基座 /admin/)
```

管理端接口统一走 `/admin-api/*`,应用端(app)接口走 `/api/*`;开发时可由 API 通过 `server.webDirs` 托管两端静态产物(见 `configs/config.example.yaml`)。
前端统一使用 Node 24.20.0（最新 LTS，见 `web/admin/.nvmrc`）；CI 使用相同版本。`npm run build` 会检查入口 JS、最大 CSS 与富文本懒加载包的体积预算，超标将失败。

## 部署

单一交付形态:**一个静态链接的 Linux 二进制 + 双端前端产物 + 一份配置**,内网交付无需 Nginx(API 自托管两端页面,访问 `http://IP:8080/` 与 `/admin/`),公网交付在前面挂 Nginx 配域名与 HTTPS。

```bash
bash deploy/binary/build-release.sh   # 产出 release/go-admin-release.tar.gz
```

照手册执行:[deploy/binary/README.md](deploy/binary/README.md)（通用 Linux：二进制 + systemd + Nginx + HTTPS；宝塔等面板场景把对应步骤换成面板操作即可）。

要点:API 默认只监听 `127.0.0.1:8080`(公网流量一律经 Nginx 反代,`server.trustedProxies` 填实际代理地址,**切勿配置为全网段**)。

**分离部署**：两端产物构建独立，可分开托管。跨域名/跨源部署时（前端域名与 API 不同源、前端直连 API），需要两步：① API 侧启用 CORS——在 `server.corsAllowedOrigins`（或 `ADMIN_SERVER_CORS_ALLOWED_ORIGINS`，逗号分隔）填写前端来源白名单；② admin 端构建时用 `VITE_PUBLIC_BASE=/` 覆盖路径基座（默认基座 `/admin/` 仅用于同域路径共存形态），并设置 `VITE_API_BASE_URL` 指向 API 完整地址。若每个前端域名各自用 Nginx 把 `/api`、`/admin-api`、`/files` 反代到 API（域内保持同源），则无需任何 CORS 配置。

### 5. 默认账号

| 账号 | 密码 | 角色 |
|---|---|---|
| admin | Admin@123456 | 超级管理员(全部权限) |
| auditor | (无默认密码,由管理员创建后分配) | 种子示例角色:仅可查看登录/操作日志 |

**首次部署后必须立即修改 admin 密码。**

## 安全边界(重要)

- **API 权限全部在服务端强制校验**(permission code,如 `system:user:create`);前端按钮隐藏只是体验,不构成安全措施。
- 密码只保存 bcrypt 哈希;登录失败返回统一文案,不区分“账号不存在/密码错误”。
- 登录失败按 用户名+IP 限流:15 分钟内失败 5 次锁定 15 分钟。
- access token(30 分钟)+ refresh token(7 天);refresh 采用轮换+吊销登记,旧刷新令牌复用会触发全量吊销。
- 退出登录、重置密码、停用账号会使用户所有已签发凭据立即失效(`token_version` 机制)。
- 审计日志对密码、Token、手机号、邮箱等字段脱敏;无法解析为 JSON 的请求体不落库;响应只记摘要/错误信息。
- **超级管理员保护**:内置账号 `admin`(id=1)不可删除、不可停用;内置角色 `super_admin` 不可删除、不可停用,保证系统永远有管理入口。
- 文件上传:扩展名+真实 MIME 双白名单、大小限制、随机文件名按日期分目录、拒绝路径穿越;公开/私有文件访问策略分离。
- 数据库密码、JWT 密钥等敏感配置只放在配置文件/环境变量,绝不入库(`sys_config` 仅存业务参数,本版未启用)。

## 数据库变更

- 一律使用 `api/migrations/` 下顺序命名的 SQL migration(`NNNNNN_name.up.sql` / `.down.sql`),内嵌进二进制,启动时自动执行未应用的 up。
- 禁止用 GORM AutoMigrate 管理生产库结构。
- 种子数据使用固定主键 + `INSERT IGNORE`,幂等且不覆盖客户对角色/菜单的修改。
- down 迁移仅供人工在明确风险下执行。

## 质量门槛

```bash
cd api && go test ./... && go vet ./...
cd web/admin && npm run lint && npm run typecheck && npm run build
```

前端工程包含 ESLint(Flat Config)+ Prettier;`npm run typecheck` 同时覆盖 src 与 vite.config.ts。

## 明确不做(v1 边界)

多租户、数据权限(仅预留)、插件/低代码、工作流、国际化、微服务/消息队列、Redis 强依赖、WebSocket、任务管理后台、MinIO/OSS、Excel 导入、安装向导、报表引擎。
