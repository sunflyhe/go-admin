#!/usr/bin/env bash
# 路线 B 发布包构建脚本:在开发机(Mac)上执行,产出可直接上传服务器的 release.tar.gz。
# 包内含:linux/amd64 二进制 + 双端前端产物 + systemd 单元 + nginx 配置 + 配置模板。
# 用法:bash deploy/binary/build-release.sh
set -euo pipefail
cd "$(dirname "$0")/../.." # 仓库根目录

OUT=release
rm -rf "$OUT"
mkdir -p "$OUT/web"

echo "== 1/4 构建 API(linux/amd64,静态链接) =="
(cd api && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o "../$OUT/go-admin" ./cmd/api)

echo "== 2/4 构建 admin 管理端 =="
(cd web/admin && npm ci && npm run build)
cp -r web/admin/dist "$OUT/web/admin"

echo "== 3/4 构建 app 应用端 =="
(cd web/app && npm ci && npm run build)
cp -r web/app/dist "$OUT/web/app"

echo "== 4/4 打包配置与模板 =="
cp deploy/binary/config.example.yaml "$OUT/config.yaml"
cp deploy/binary/nginx.conf "$OUT/"
tar -czf "$OUT/go-admin-release.tar.gz" -C "$OUT" .

echo
echo "完成:release/go-admin-release.tar.gz"
echo "上传:scp release/go-admin-release.tar.gz root@<服务器>:/opt/"
