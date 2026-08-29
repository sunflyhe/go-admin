# api — Go 后端 API

Gin + GORM + MySQL 8 的单二进制后端 API。本 README 面向在 `api/` 内开发的工程师，项目整体定位与部署见仓库根 [README](../README.md)。

## 目录结构

```text
api/
├── cmd/api/main.go      启动入口:配置 → 日志 → DB → 迁移 → HTTP → 优雅关闭
├── internal/            私有代码(编译器禁止外部模块导入)
│   ├── handler/         HTTP 处理器:参数绑定、Header/IP/UA 读取、统一响应
│   │   └── *_dto.go     HTTP 绑定 DTO(带 binding 标签)+ toInput() 显式转换
│   ├── service/         业务逻辑:不依赖 Gin,方法接收 context.Context
│   ├── model/           GORM 模型(仅查询映射,表结构以 migrations 为准)
│   ├── middleware/      请求 ID、访问日志、Panic 恢复、认证/权限、审计记录器
│   ├── config/          配置加载(文件 + 环境变量覆盖 + 启动校验)
│   └── router/          路由注册与服务装配(唯一的 DI 点)
├── pkg/                 可复用通用包:auth(JWT)、database、logger、migrate、
│                        errs(错误码)、resp(统一响应)、page(分页)、validate
├── migrations/          SQL migration(NNNNNN_name.up/down.sql,embed 进二进制)
├── test/testutil/       测试基础设施(内存 SQLite + 种子数据)
└── configs/             config.example.yaml(真实 config.yaml 不入库)
```

## 分层规则

依赖方向:`handler → service → model`,公共能力向下引用 `pkg/`。

1. **Handler 是唯一依赖 Gin 的 HTTP 层**:负责 `ShouldBindJSON/Query`、路径参数、multipart、Header/IP/UA 读取与统一响应;调用 Service 时使用 `c.Request.Context()`。
2. **Service 不 import gin**:方法接收 `context.Context`,数据库统一 `db.WithContext(ctx)`;需要登录态时接收 `service.Actor`,需要 IP/UA 时接收 `service.LoginMeta`,由 Handler 显式转换传入——因此 Service 可被 CLI、定时任务、消息消费直接复用。
3. **DTO 分层**:HTTP 绑定 DTO 在 `handler/*_dto.go`(带 binding 标签);Service 输入 DTO 定义在对应 service 文件(无标签,按域前缀命名如 `UserSaveInput`);输出 VO(如 `UserItem`)由 Service 返回,不暴露 `model.SysUser` 密码字段等内部结构。
4. **新模块套路**:model 进 `internal/model` → 业务进 `internal/service` → 接口进 `internal/handler` → 路由在 `internal/router/router.go` 注册(同时声明权限码)。

## 快速开始

```bash
# 1. 建库(空库即可,启动时自动迁移+种子)
mysql -uroot -p -e "CREATE DATABASE go_admin DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 2. 配置
cp configs/config.example.yaml configs/config.yaml
#    填写 mysql.dsn 与 jwt.secret(≥16 位,建议 32 位随机串)

# 3. 运行
go run ./cmd/api -config configs/config.yaml

# 4. 验证
curl http://localhost:8080/healthz   # {"status":"ok"}
curl http://localhost:8080/readyz    # {"status":"ready"}(检查 DB)
```

所有配置项可用环境变量覆盖(优先级高于文件):`ADMIN_MYSQL_DSN`、`ADMIN_JWT_SECRET`、`ADMIN_SERVER_ADDR`、`ADMIN_UPLOAD_DIR`、`ADMIN_AUDIT_RETENTION_DAYS` 等,完整清单见 `config/config.go` 的 `applyEnv`。

默认账号 `admin / Admin@123456`(种子幂等写入,首次部署必须改密)。

## 数据库迁移

- 库结构一律通过 `migrations/` 的 SQL 文件管理:**禁止 GORM AutoMigrate**。
- 文件命名 `NNNNNN_name.up.sql` / `.down.sql`,通过 `migrations/embed.go` 内嵌,服务启动自动执行未应用的 up(空库一键初始化)。
- 种子数据(超管账号、内置角色、菜单树)固定主键 + `INSERT IGNORE`,幂等且不覆盖客户修改。
- down 迁移仅供人工在明确风险下执行,代码不提供自动路径。

## 认证与安全要点

- access token(默认 30 分钟)+ refresh token(默认 7 天);refresh 轮换 + 吊销登记,并发刷新由条件更新保证只有一次成功;旧令牌复用触发全量吊销。
- 退出、重置密码、停用账号、改角色都会自增 `token_version`,已签发凭据立即失效。
- 登录失败按 用户名+IP 限流(15 分钟内 5 次,锁 15 分钟);登录日志记录 IP/UA(来自 `LoginMeta`)。
- 超管保护:`admin`(id=1)与 `super_admin` 角色不可删停,代码在 service 层强制(`checkStatusChange`、`Delete` 等)。
- 权限码(如 `system:user:create`)在路由注册处声明,由 `middleware.Authn.RequirePerm` 服务端强制校验;前端隐藏按钮仅为体验。
- 文件上传:大小上限(Handler MaxBytesReader)+ 扩展名白名单 + 真实 MIME 内容嗅探(Service),随机文件名按日期分目录;公开/私有文件访问策略分离,公开文件输出前仍校验 DB 的 `is_public` 标记。

## 测试

```bash
go test ./...        # 全量
go vet ./...         # 静态检查
```

- 测试库用纯 Go SQLite(`test/testutil`,按测试名隔离的内存库),无需外部 MySQL;生产表结构仍由 SQL migration 管理,测试库允许 AutoMigrate。
- Service 测试直接 `context.Background()` 调用,不构造 Gin(见 `internal/service/auth_test.go`、`file_test.go`)。
- Handler/路由层有 httptest 集成测试(`internal/handler/auth_test.go`、`internal/router/router_test.go`),覆盖登录、刷新轮换、登出失效、两角色权限差异。
