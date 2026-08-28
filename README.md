# Go Admin — Go 企业后台开发底座

可复用的企业后台开发底座:登录认证、账号/角色/菜单 RBAC、操作审计、登录日志、本地文件上传、Excel 导出。
定位是**新业务模块的起点**,不是低代码平台,也不是 SaaS。优先级:稳定可维护 > 安全边界清楚 > 私有化易部署 > 功能数量。

技术栈:Go + Gin + GORM(仅查询映射)+ MySQL 8(golang-migrate 管理 schema)+ JWT(bcrypt 密码)+ Vue 3 + TypeScript + Element Plus + Pinia。

## 目录结构

后端目录布局**参照 Hyperf 的习惯设计**(bin/、config/、app/、migrations/),从 Hyperf 迁移过来的开发者可以直接对号入座:

```text
server/                  Go 后端(单二进制)
├── bin/                 启动入口 main.go             ≈ bin/hyperf.php
├── app/
│   ├── controller/      HTTP 控制器(参数绑定+响应)   ≈ app/Controller
│   ├── service/         业务逻辑 + 领域服务            ≈ app/Service
│   ├── model/           GORM 数据模型                 ≈ app/Model
│   └── middleware/      请求 ID/访问日志/恢复/认证/权限  ≈ app/Middleware
├── config/              配置加载代码 + config.example.yaml ≈ config/(autoload)
├── routes/              路由注册                      ≈ config/routes.php
├── pkg/                 无业务语义的通用包(JWT、DB、日志、迁移、错误码、响应、分页、校验)
├── migrations/          SQL migration(embed,启动时自动执行 up)≈ migrations/
├── test/                测试基础设施                  ≈ test/
└── config/config.example.yaml
web/       Vue3 管理端
deploy/    Dockerfile 与 docker-compose 示例
docs/      OpenAPI 文档
```

与 Hyperf 的差异:Go 无注解与自动 DI,服务在 `routes/` 中手工装配;请求 DTO 定义在各 service 文件内(相当于 FormRequest 内联);`pkg/` 对应"框架级组件"。

## 快速开始(本地)

### 1. 准备 MySQL

```bash
mysql -uroot -p -e "CREATE DATABASE go_admin DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
```

### 2. 配置

```bash
cd server
cp config/config.example.yaml config/config.yaml
# 编辑 config/config.yaml:填写 mysql.dsn 与 jwt.secret(至少 16 位,建议 32 位随机串)
```

或完全用环境变量(优先级高于文件):`ADMIN_MYSQL_DSN`、`ADMIN_JWT_SECRET`、`ADMIN_SERVER_ADDR` 等,见 `config/config.example.yaml` 注释。

### 3. 启动(空库自动迁移 + 种子)

```bash
cd server
go run ./bin -config config/config.yaml
```

服务启动时会自动执行未应用的迁移与幂等种子数据;也可用 `server/migrations/` 下的 SQL 手工执行。

### 4. 前端

```bash
cd web
npm install
npm run dev        # 开发模式,代理到 localhost:8080
npm run build      # 生产构建,产物在 web/dist
```

生产部署时把 `server/config/config.yaml` 的 `server.webDir` 指向 `web/dist` 即可由后端单二进制托管前端;或用 Nginx 单独托管前端静态文件。

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

- 一律使用 `server/migrations/` 下顺序命名的 SQL migration(`NNNNNN_name.up.sql` / `.down.sql`),内嵌进二进制,启动时自动执行未应用的 up。
- 禁止用 GORM AutoMigrate 管理生产库结构。
- 种子数据使用固定主键 + `INSERT IGNORE`,幂等且不覆盖客户对角色/菜单的修改。
- down 迁移仅供人工在明确风险下执行。

## 质量门槛

```bash
cd server && go test ./... && go vet ./...
cd web && npm run lint && npm run typecheck && npm run build
```

前端工程包含 ESLint(Flat Config)+ Prettier;`npm run typecheck` 同时覆盖 src 与 vite.config.ts。

## API 文档

见 [docs/openapi.yaml](docs/openapi.yaml)。统一响应格式:

```json
{"code": 0, "message": "success", "data": {}}
```

分页响应 `data`:`{"list":[],"total":0,"page":1,"pageSize":20}`。

## 明确不做(v1 边界)

多租户、数据权限(仅预留)、插件/低代码、工作流、国际化、微服务/消息队列、Redis 强依赖、WebSocket、任务管理后台、MinIO/OSS、Excel 导入、安装向导、报表引擎。启用条件见执行说明文档第 9 节。
