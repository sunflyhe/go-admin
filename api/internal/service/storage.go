// Package file 文件域:统一 Storage 抽象 + 第一版 Local 实现。
// 安全规则:
//   - 扩展名与真实 MIME 双重白名单校验
//   - 随机文件名 + 日期目录,存储路径由服务端生成,不使用用户输入路径
//   - 公开文件经静态前缀访问;私有文件走鉴权下载接口
package service

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Storage 存储抽象。第一版仅实现 Local;后续可扩展 MinIO/S3/OSS,业务层不感知差异。
type Storage interface {
	Save(relPath string, r io.Reader) (err error)
	Open(relPath string) (*os.File, error)
	Delete(relPath string) error
	Stat(relPath string) (int64, error)
}

// Local 本地磁盘存储,所有路径都限制在 baseDir 内。
type Local struct{ baseDir string }

func NewLocal(baseDir string) (*Local, error) {
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}
	return &Local{baseDir: abs}, nil
}

// resolve 将相对路径转换为绝对路径,并拒绝任何包含 ".." 的路径穿越尝试。
func (l *Local) resolve(relPath string) (string, error) {
	if strings.Contains(filepath.ToSlash(relPath), "..") {
		return "", os.ErrInvalid
	}
	clean := filepath.Clean("/" + strings.TrimPrefix(relPath, "/")) // 以 / 为根做规范化
	abs := filepath.Join(l.baseDir, clean)
	if abs != l.baseDir && !strings.HasPrefix(abs, l.baseDir+string(filepath.Separator)) {
		return "", os.ErrInvalid
	}
	return abs, nil
}

func (l *Local) Save(relPath string, r io.Reader) error {
	abs, err := l.resolve(relPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(abs, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func (l *Local) Open(relPath string) (*os.File, error) {
	abs, err := l.resolve(relPath)
	if err != nil {
		return nil, err
	}
	return os.Open(abs)
}

func (l *Local) Delete(relPath string) error {
	abs, err := l.resolve(relPath)
	if err != nil {
		return err
	}
	return os.Remove(abs)
}

func (l *Local) Stat(relPath string) (int64, error) {
	abs, err := l.resolve(relPath)
	if err != nil {
		return 0, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// BaseDir 返回存储根目录(用于静态文件服务)。
func (l *Local) BaseDir() string { return l.baseDir }
