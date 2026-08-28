// Package migrations 通过 embed 内嵌 SQL 迁移文件,由 platform/migrate 在启动时执行。
package migrations

import "embed"

// FS 内嵌全部 .sql 迁移文件(文件名需符合 golang-migrate 的 NNNNNN_name.up/down.sql 规范)。
//
//go:embed *.sql
var FS embed.FS
