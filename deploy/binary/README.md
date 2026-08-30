# 部署手册(二进制 + systemd + Nginx)

目标机 Linux x86_64(其他架构改 `build-release.sh` 的 `GOARCH`);安装目录 `/opt/go-admin`。宝塔等面板场景:数据库/守护进程/证书改用面板功能,步骤一一对应。

## 构建(开发机)

```bash
bash deploy/binary/build-release.sh
scp release/go-admin-release.tar.gz root@<服务器>:/opt/
```

## 服务器

```bash
apt install -y mysql-server nginx
systemctl enable --now mysql nginx

mysql -e "
CREATE DATABASE go_admin DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'goadmin'@'localhost' IDENTIFIED BY '<密码>';
GRANT ALL PRIVILEGES ON go_admin.* TO 'goadmin'@'localhost';
FLUSH PRIVILEGES;"

cd /opt/go-admin && tar -xzf /opt/go-admin-release.tar.gz && chmod +x go-admin
useradd -r -s /usr/sbin/nologin goadmin && chown -R goadmin:goadmin /opt/go-admin

# 改 config.yaml 两处:dsn 密码、jwt.secret(openssl rand -base64 32)
vim config.yaml
```

## systemd

```bash
cat > /etc/systemd/system/go-admin.service <<'EOF'
[Unit]
Description=Go Admin API
After=network-online.target mysql.service
Wants=network-online.target

[Service]
Type=simple
User=goadmin
Group=goadmin
WorkingDirectory=/opt/go-admin
ExecStart=/opt/go-admin/go-admin -config /opt/go-admin/config.yaml
Restart=on-failure
RestartSec=3
NoNewPrivileges=true
ProtectSystem=full
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/opt/go-admin

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload && systemctl enable --now go-admin
curl -s http://127.0.0.1:8080/healthz
```

## Nginx

```bash
cp nginx.conf /etc/nginx/sites-available/go-admin
sed -i 's/server_name _;/server_name <域名>/' /etc/nginx/sites-available/go-admin
ln -sf /etc/nginx/sites-available/go-admin /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default
nginx -t && systemctl reload nginx

apt install -y certbot python3-certbot-nginx && certbot --nginx -d <域名>
```

## 端口

安全组/ufw 只放行 22、80、443;**8080 仅本机**(config 的 `addr` 为 `127.0.0.1:8080`)。

## 验证

浏览器访问 `/`(app)与 `/admin/`(管理端,默认 `admin / Admin@123456`,首登改密)。
502 → `systemctl status go-admin`;日志 IP 全是 127.0.0.1 → 检查 `trustedProxies`。

## 升级

重新构建上传覆盖(**保留 config.yaml**)→ `systemctl restart go-admin`;迁移自动增量执行。
