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

其余项(addr 127.0.0.1:8080、trustedProxies、webDirs)保持模板默认即可。

## 5. 进程守护(让 API 常驻)

【软件商店】→ 搜索安装 **进程守护管理器**(Supervisor),然后:

添加守护进程:

| 项 | 值 |
|---|---|
| 名称 | go-admin |
| 启动用户 | www |
| 运行目录 | /www/wwwroot/go-admin |
| 启动命令 | /www/wwwroot/go-admin/go-admin -config /www/wwwroot/go-admin/config.yaml |
| 进程数量 | 1 |

启动后点【日志】确认看到 `HTTP 服务启动`。验证:`curl 127.0.0.1:8080/healthz` 返回 ok。

> 偏好 systemd 的话也可以用 `deploy/binary/go-admin.service`(改 User/路径后照标准手册装),与守护管理器二选一,别同时跑两个。

## 6. 添加站点与反向代理

【网站】→【添加站点】:

- 域名:你的域名(先去 DNS 加 A 记录指向服务器 IP)
- 根目录:任选(如 /www/wwwroot/go-admin-site,我们不用它托管静态)
- PHP 版本:**纯静态**

创建后进入站点【设置】→【配置文件】,在 `server {}` 内**追加**以下 location 块(保留宝塔生成的 server_name/SSL/日志行):

```nginx
  location /api/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
  }
  location /admin-api/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
  }
  location /files/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
  }
  location /admin/assets/ {
    alias /www/wwwroot/go-admin/web/admin/assets/;
    expires 1y;
    add_header Cache-Control "public, immutable";
  }
  location /admin/ {
    alias /www/wwwroot/go-admin/web/admin/;
    expires -1;
    try_files $uri $uri/ /admin/index.html;
  }
  location /assets/ {
    root /www/wwwroot/go-admin/web/app;
    try_files $uri =404;
    expires 1y;
    add_header Cache-Control "public, immutable";
  }
  location / {
    root /www/wwwroot/go-admin/web/app;
    expires -1;
    try_files $uri $uri/ /index.html;
  }
```

保存前把站点原有的 `location /`(宝塔默认生成的那个)删掉或注释,避免冲突。保存后宝塔自动 `nginx -t` 并 reload。

## 7. SSL

站点【设置】→【SSL】→【Let's Encrypt】→ 勾选域名 → 申请,成功后开启【强制 HTTPS】。证书续期由宝塔自动处理。

## 8. 端口

- 阿里云安全组 + 宝塔【安全】页:放行 80、443(SSH 22 已有);**8080 不放行**
- 数据库权限保持"本地服务器",不开外网访问

## 9. 验收

- `https://你的域名/` → app 端
- `https://你的域名/admin/` → 管理端,`admin / Admin@123456`,**登录后立即改密**
- 管理端【日志管理】里看到刚才的登录记录,且客户端 IP 是你的真实 IP(不是 127.0.0.1)

## 升级版本

1. 本机重新 `bash deploy/binary/build-release.sh`,上传覆盖 `/www/wwwroot/go-admin/`(config.yaml 是你自己改过的,**覆盖时保留它**,或上传到临时目录后只替换 go-admin、web/ 两个产物)
2. 守护管理器里【重启】go-admin——数据库迁移在启动时自动增量执行
