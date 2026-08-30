# 路线 B 部署手册(二进制 + systemd + Nginx)

目标机:Ubuntu 24.04 x86_64(已按此架构出产物;其他架构改 `build-release.sh` 里的 `GOARCH`)。
约定安装目录:`/opt/go-admin`。全程假设你以 root 操作。

## 0. 本机构建发布包

```bash
bash deploy/binary/build-release.sh
scp release/go-admin-release.tar.gz root@<服务器IP>:/opt/
```

## 1. 服务器:装依赖

```bash
apt update && apt install -y mysql-server nginx
systemctl enable --now mysql nginx
mysql_secure_installation   # 按提示做基础加固
```

## 2. 建库建账号(不要用 root 跑应用)

```bash
mysql <<'SQL'
CREATE DATABASE go_admin DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'goadmin'@'localhost' IDENTIFIED BY '换成强密码';
GRANT ALL PRIVILEGES ON go_admin.* TO 'goadmin'@'localhost';
FLUSH PRIVILEGES;
SQL
```

## 3. 解压与配置

```bash
mkdir -p /opt/go-admin && cd /opt/go-admin
tar -xzf /opt/go-admin-release.tar.gz
chmod +x go-admin

# 编辑 config.yaml:两处 TODO(dsn 密码、jwt.secret)
#   jwt.secret 用:openssl rand -base64 32
vim config.yaml

# 应用运行账号与目录归属
useradd -r -s /usr/sbin/nologin goadmin
chown -R goadmin:goadmin /opt/go-admin
sudo -u goadmin ./go-admin -config config.yaml   # 手动试跑一次,看到 "HTTP 服务启动" 即成,Ctrl+C
```

## 4. systemd 托管 API

```bash
cp go-admin.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now go-admin
systemctl status go-admin            # active (running)
curl -s http://127.0.0.1:8080/healthz   # {"status":"ok"}
```

## 5. Nginx 站点

```bash
cp nginx.conf /etc/nginx/sites-available/go-admin
vim /etc/nginx/sites-available/go-admin   # 改 server_name 为你的域名
ln -sf /etc/nginx/sites-available/go-admin /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default
nginx -t && systemctl reload nginx
```

## 6. 防火墙/安全组

- 阿里云安全组:只放行 22、80、443;**8080 绝不对外**
- 本机如启用 ufw:同样只放行 80/443

## 7. HTTPS(有域名时)

```bash
apt install -y certbot python3-certbot-nginx
certbot --nginx -d <你的域名>    # 自动改写配置加 443 与 80→443 跳转,证书自动续期
```

## 8. 验收

- `http://<域名或IP>/` → app 端
- `http://<域名或IP>/admin/` → 管理端,默认账号 `admin / Admin@123456`,**登录后立即改密**
- 数据库迁移已自动执行(`migration done` 字样出现在 `journalctl -u go-admin`)

## 升级版本

```bash
# 本机:重新 build-release.sh + scp
# 服务器:
cd /opt/go-admin && tar -xzf /opt/go-admin-release.tar.gz && chown -R goadmin:goadmin /opt/go-admin
systemctl restart go-admin   # 数据库迁移在启动时自动增量执行
```

## 常见问题

| 现象 | 排查 |
|---|---|
| 起不来 `bind: address already in use` | 8080 被占:`ss -ltnp \| grep 8080` |
| 页面 502 | `systemctl status go-admin`;API 没起来或监听地址不是 127.0.0.1 |
| 登录接口 502 但页面正常 | 同上,看 `journalctl -u go-admin -f` |
| 客户端 IP 全是 127.0.0.1 | config 的 `trustedProxies` 必须含 `127.0.0.1` |
| 数据库连接拒绝 | MySQL 未启动,或 dsn 密码与第 2 步不一致 |
