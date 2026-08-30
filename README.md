# Go Admin

一个面向私有化部署的 Go 企业后台开发底座。它提供可直接复用的认证、RBAC、审计、文件与内容管理能力，适合作为中小型业务系统的起点，而非低代码平台或多租户 SaaS。

项目优先级固定为：**稳定可维护 > 安全边界清楚 > 私有化易部署 > 功能数量**。

## 已具备的能力

- 管理端登录、个人资料、密码与头像管理
- 用户、角色、菜单与按钮权限（服务端 RBAC）
- 登录日志与脱敏操作审计
- 本地文件中心：分组、公开/私有文件、鉴权下载、上传安全校验
- 系统参数、数据字典、文章分类与富文本文章管理
- 用户导出与数据库 migration、幂等初始化种子
- Vue 3 管理端及最小化 app 端首页；后端可选托管两端静态产物

## 技术栈

| 层级 | 选型 |
| --- | --- |
| 后端 | Go、Gin、GORM、MySQL 8、golang-migrate、JWT、bcrypt |
| 前端 | Vue 3、TypeScript、Vite、Element Plus、Pinia |
| 交付 | Linux 静态二进制、systemd、Nginx（可选） |

## 项目结构

```text
api/                         Go API（单二进制）
├── cmd/api/                 启动入口与依赖装配
├── internal/
│   ├── handler/             HTTP 边界：参数绑定、DTO 转换、统一响应
│   ├── service/             不依赖 Gin 的业务逻辑
│   ├── model/               GORM 模型
│   ├── middleware/          认证、权限、审计、日志与 CORS
│   ├── config/              配置加载
│   └── router/              引擎、依赖装配与分端路由注册
├── migrations/              顺序 SQL migration（编译进二进制）
├── pkg/                     无业务语义的可复用组件
├── configs/                 配置示例
└── test/                    测试工具
web/
├── admin/                   Vue 管理端，页面路径 /admin/
└── app/                     Vue app 端，页面路径 /
deploy/binary/               发布包构建脚本、systemd/Nginx 模板与部署手册
```

接口按端隔离：管理端为 `/admin-api/*`，app/开放端为 `/api/*`。管理端权限码不复用于 app 端。

## 本地运行

### 前置条件

- Go（版本以 [`api/go.mod`](api/go.mod) 为准）
- MySQL 8
- Node.js `24.20.0` 与 npm 10+（`web/admin/.nvmrc`、`web/app/.nvmrc` 已锁定版本）

### 1. 创建数据库并配置后端

```bash
mysql -uroot -p -e "CREATE DATABASE go_admin DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

cd api
cp configs/config.example.yaml configs/config.yaml
```

编辑 `api/configs/config.yaml`，至少填写：

- `mysql.dsn`
- `jwt.secret`：不少于 16 位；生产建议使用 32 位以上随机值

环境变量可覆盖配置文件，例如 `ADMIN_MYSQL_DSN`、`ADMIN_JWT_SECRET`、`ADMIN_SERVER_ADDR`。完整选项和说明见 [`api/configs/config.example.yaml`](api/configs/config.example.yaml)。

### 2. 启动 API

```bash
cd api
go run ./cmd/api -config configs/config.yaml
```

首次连接空库时，服务会自动执行尚未应用的 up migration 与幂等种子数据。健康检查地址：

```text
GET /healthz    # 进程存活
GET /readyz     # MySQL 可用
```

开发时可安装 [Air](https://github.com/air-verse/air) 进行热重载：

```bash
go install github.com/air-verse/air@latest
cd api && air
```

### 3. 启动前端

分别在两个终端执行：

```bash
cd web/admin
nvm use
npm install
npm run dev
```

管理端访问 `http://localhost:5173/admin/`，开发服务器会将 `/admin-api` 代理到 `localhost:8080`。

```bash
cd web/app
nvm use
npm install
npm run dev
```

app 端目前提供最小化首页，作为后续 app 业务的独立入口。生产构建使用各端的 `npm run build`；管理端构建还会执行资源体积预算检查。

### 默认账号

| 账号 | 密码 | 说明 |
| --- | --- | --- |
| `admin` | `Admin@123456` | 内置超级管理员 |
| `auditor` | 不设默认密码 | 仅作为审计角色种子示例 |

首次部署后请立即修改 `admin` 密码。

## 部署

标准交付物为一个 Linux 二进制、两端前端产物和一份运行配置：

```bash
bash deploy/binary/build-release.sh
```

脚本会生成 `release/go-admin-release.tar.gz`。Linux x86_64 的 systemd、Nginx、HTTPS、升级与验证步骤见 [`deploy/binary/README.md`](deploy/binary/README.md)。

可以由 API 直接托管两端静态资源：在 `server.webDirs` 配置 app 与 admin 的构建目录；也可交由 Nginx 或其他静态服务托管。公网部署建议由 Nginx 终止 HTTPS 并反向代理 API。

部署在反向代理之后时，务必将 `server.trustedProxies`（或 `ADMIN_SERVER_TRUSTED_PROXIES`）限制为实际代理 IP/CIDR；默认不信任 `X-Forwarded-For`。前后端跨源直连时，再通过 `server.corsAllowedOrigins` 配置明确来源白名单。

## 安全与数据约束

- 所有受保护管理接口均在服务端按 permission code 校验；前端隐藏按钮不是安全控制。
- 密码只保存 bcrypt 哈希；刷新令牌使用轮换、吊销登记与条件更新；停用、改密、登出和角色变更会使既有凭据及时失效。
- 审计日志对敏感字段脱敏，不记录密码、令牌或密钥明文。
- 上传执行文件大小、扩展名、真实 MIME 与路径穿越校验；私有文件只能经鉴权下载，公开文件也必须由服务端确认 `is_public`。
- 内置 `admin` 账号和 `super_admin` 角色受到保护，避免系统失去管理入口。
- 数据库结构仅通过新增 `api/migrations/` SQL migration 变更，禁止 GORM `AutoMigrate`；种子必须幂等且不得覆盖客户配置。

## 开发约定

- Handler 只处理 HTTP/Gin 边界；HTTP DTO 放在 `internal/handler/*_dto.go`，显式转换后以 `context.Context` 调用 Service。
- Service 不依赖 Gin，数据库操作使用 `db.WithContext(ctx)`；不引入泛化 Repository、DDD/CQRS 或依赖注入框架。
- 小型业务模块通常新增 `internal/handler/<domain>.go`、`internal/service/<domain>.go` 和相应 model，并在所属端的路由注册文件线性追加路由。
- 新增数据库字段或表时只增加顺序 migration，兼顾空库初始化与已有数据升级。

详细的架构与安全不可回退项见 [`AGENTS.md`](AGENTS.md)。

## 质量检查

提交前至少执行：

```bash
cd api && go test ./... && go vet ./...
cd web/admin && npm run lint && npm run typecheck && npm run build
cd web/app && npm run lint && npm run typecheck && npm run build
git diff --check
```

涉及浏览器交互、真实 MySQL、Docker 或生产环境的变更，还应进行对应环境验证；静态检查与构建成功不等同于真实环境验收。

