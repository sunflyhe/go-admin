# 路线 B 部署手册(宝塔面板版)

适用于服务器已装宝塔面板的场景。与标准版手册的差异:**MySQL 在面板建、进程用宝塔守护管理器、SSL 面板一键签**,全程基本点鼠标。

前置确认:宝塔【软件商店】里已安装 **Nginx** 和 **MySQL**(8.0 为佳)。

## 1. 本机构建发布包(与标准版相同)

```bash
bash deploy/binary/build-release.sh
```

得到 `release/go-admin-release.tar.gz`,通过宝塔【文件】页面上传到 `/www/wwwroot/`,或本机 scp。

## 2. 解压发布包

宝塔【文件】→ 进入 `/www/wwwroot/` → 对 tar.gz 点【解压】到 `/www/wwwroot/go-admin`。
(或 SSH:`mkdir -p /www/wwwroot/go-admin && tar -xzf /www/wwwroot/go-admin-release.tar.gz -C /www/wwwroot/go-admin`)

```bash
chmod +x /www/wwwroot/go-admin/go-admin
```

## 3. 面板建数据库

【数据库】→【添加数据库】:数据库名 `go_admin`、用户名 `goadmin`(或面板自动生成)、密码用面板生成的强密码,访问权限选**本地服务器**。

## 4. 改配置(两处 TODO)

编辑 `/www/wwwroot/go-admin/config.yaml`:

- `mysql.dsn`:把 `CHANGE_ME` 换成上一步的密码(主机保持 `127.0.0.1:3306`)
- `jwt.secret`:终端 `openssl rand -base64 32` 生成后粘贴

其余项(addr 127.0.0.1:8080、trustedProxies)保持模板默认即可。**webDirs 与 upload.dir 建议改成绝对路径**(Go 项目的运行目录不保证是二进制所在目录):

```yaml
  webDirs:
    app: "/www/wwwroot/go-admin/web/app"
    admin: "/www/wwwroot/go-admin/web/admin"
```
```yaml
upload:
  dir: "/www/wwwroot/go-admin/data/uploads"
```

改完执行 `chown -R www:www /www/wwwroot/go-admin`,让 www 用户可读写。

## 5. 用【Go 项目】让 API 常驻(推荐)

面板【网站】→【Go 项目】→【添加 Go 项目】,按下表填写:

| 字段 | 值 | 说明 |
|---|---|---|
| 项目执行文件 | /www/wwwroot/go-admin/go-admin | 解压出的二进制 |
| 项目名称 | go-admin | |
| 项目端口 | 8080 | API 真实监听端口 |
| 放行端口 | **不勾** | 8080 只给 Nginx 反代,勾了会把 API 暴露到公网 |
| 执行命令 | -config /www/wwwroot/go-admin/config.yaml | 启动参数 |
| 环境变量 | 无 | 配置都在 config.yaml |
| 运行用户 | www | |
| 开机启动 | 勾选 | |
| 绑定域名 | 你的域名 | DNS 先解析到服务器 IP;填了宝塔会自动生成反代站点 |

确定后【日志】里确认 `HTTP 服务启动`;`curl 127.0.0.1:8080/healthz` 返回 ok。

因为 Go 后端通过 `webDirs` 自己托管两端静态页面,**绑定域名生成的反代站点(全站代理到 8080)开箱即用**,不需要任何自定义 location。

> 替代方案:【软件商店】的进程守护管理器(Supervisor),或 `deploy/binary/go-admin.service`(systemd)。三选一,不要同时跑两个。

## 6. 站点微调(仅在使用第 5 节"绑定域名"时基本无需改动)

绑定域名自动生成的站点会把所有请求代理到 8080,Go 端已能处理全部路径。若想静态资源走 Nginx 直出(可选优化),再进站点【配置文件】追加:

```nginx
  location /admin/assets/ {
    alias /www/wwwroot/go-admin/web/admin/assets/;
    expires 1y;
    add_header Cache-Control "public, immutable";
  }
  location /assets/ {
    root /www/wwwroot/go-admin/web/app;
    expires 1y;
    add_header Cache-Control "public, immutable";
  }
```

不加也完全不影响功能——静态文件由 Go 输出,带正确的 no-cache/缓存语义。

## 7. SSL

站点(【Go 项目】绑定域名自动生成的那个站点)【设置】→【SSL】→【Let's Encrypt】→ 勾选域名 → 申请,成功后开启【强制 HTTPS】。证书续期由宝塔自动处理。

## 8. 端口

- 阿里云安全组 + 宝塔【安全】页:放行 80、443(SSH 22 已有);**8080 不放行**
- 数据库权限保持"本地服务器",不开外网访问

## 9. 验收

- `https://你的域名/` → app 端
- `https://你的域名/admin/` → 管理端,`admin / Admin@123456`,**登录后立即改密**
- 管理端【日志管理】里看到刚才的登录记录,且客户端 IP 是你的真实 IP(不是 127.0.0.1)

## 升级版本

1. 本机重新 `bash deploy/binary/build-release.sh`,上传覆盖 `/www/wwwroot/go-admin/`(config.yaml 是你自己改过的,**覆盖时保留它**,或上传到临时目录后只替换 go-admin、web/ 两个产物)
2. 【Go 项目】列表里【重启】go-admin——数据库迁移在启动时自动增量执行
