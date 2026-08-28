package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalSaveOpenDelete(t *testing.T) {
	dir := t.TempDir()
	l, err := NewLocal(dir)
	if err != nil {
		t.Fatal(err)
	}
	rel := "2026/08/28/abc.txt"
	if err := l.Save(rel, strings.NewReader("hello world")); err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	f, err := l.Open(rel)
	if err != nil {
		t.Fatalf("打开失败: %v", err)
	}
	defer f.Close()
	buf := make([]byte, 64)
	n, _ := f.Read(buf)
	if string(buf[:n]) != "hello world" {
		t.Fatalf("内容不匹配: %s", buf[:n])
	}
	size, err := l.Stat(rel)
	if err != nil || size != 11 {
		t.Fatalf("Stat 不匹配: %d %v", size, err)
	}
	if err := l.Delete(rel); err != nil {
		t.Fatalf("删除失败: %v", err)
	}
}

func TestLocalRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	l, _ := NewLocal(dir)
	for _, p := range []string{"../../etc/passwd", "../outside.txt", "/../../etc/hosts"} {
		if err := l.Save(p, strings.NewReader("x")); err == nil {
			t.Fatalf("路径穿越未被拦截: %s", p)
		}
	}
	// 确认没有文件被写到存储目录之外
	entries, _ := os.ReadDir(filepath.Dir(dir))
	for _, e := range entries {
		if e.Name() == filepath.Base(dir) {
			continue
		}
		if strings.Contains(e.Name(), "outside") {
			t.Fatal("文件被写到存储目录之外")
		}
	}
}
