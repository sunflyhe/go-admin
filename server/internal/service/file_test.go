// 文件服务测试:不构造 Gin 上下文,以文件流直接调用,
// 验证扩展名白名单、真实 MIME 嗅探校验与上传人记录不因脱离 HTTP 而回退。
package service

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/hesunfly/hesunfly-admin-go/server/test/testutil"
)

func newFileService(t *testing.T) (*FileService, context.Context) {
	t.Helper()
	storage, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := NewFileService(testutil.NewTestDB(t), storage, 5)
	return svc, context.Background()
}

// pngMagic PNG 文件头,DetectContentType 据此识别为 image/png。
var pngMagic = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

func TestUploadWithoutGinSucceeds(t *testing.T) {
	svc, ctx := newFileService(t)
	actor := Actor{ID: 1, Username: "admin", IsSuper: true}
	content := append(append([]byte{}, pngMagic...), bytes.Repeat([]byte{0}, 32)...)

	result, err := svc.Upload(ctx, actor, &FileUploadInput{
		FileName: "avatar.png",
		Size:     int64(len(content)),
		Content:  bytes.NewReader(content),
		IsPublic: false,
	})
	if err != nil {
		t.Fatalf("上传失败: %v", err)
	}
	if result.MIME != "image/png" {
		t.Fatalf("应按内容嗅探为 image/png: %s", result.MIME)
	}
	if !strings.Contains(result.StorePath, ".png") {
		t.Fatalf("存储路径应保留扩展名: %s", result.StorePath)
	}
	var entry = *result
	_ = entry
}

func TestUploadRejectsDisallowedExtension(t *testing.T) {
	svc, ctx := newFileService(t)
	actor := Actor{ID: 1, Username: "admin", IsSuper: true}
	content := append(append([]byte{}, pngMagic...), bytes.Repeat([]byte{0}, 32)...)

	// 内容是 PNG 但扩展名不在白名单:拒绝
	_, err := svc.Upload(ctx, actor, &FileUploadInput{
		FileName: "payload.exe", Size: int64(len(content)),
		Content: bytes.NewReader(content),
	})
	if err == nil {
		t.Fatal("扩展名不在白名单应拒绝")
	}
}

func TestUploadRejectsDisallowedRealMIME(t *testing.T) {
	svc, ctx := newFileService(t)
	actor := Actor{ID: 1, Username: "admin", IsSuper: true}

	// 扩展名伪装为 .png,真实内容是二进制数据:按内容嗅探拒绝
	binary := bytes.Repeat([]byte{0x00, 0x01, 0x02, 0x03}, 128)
	_, err := svc.Upload(ctx, actor, &FileUploadInput{
		FileName: "fake.png", Size: int64(len(binary)),
		Content: bytes.NewReader(binary),
	})
	if err == nil {
		t.Fatal("真实 MIME 不在白名单应拒绝")
	}
}

func TestUploadRecordsActor(t *testing.T) {
	svc, ctx := newFileService(t)
	actor := Actor{ID: 7, Username: "carol", IsSuper: false}
	content := strings.NewReader("hello world")

	result, err := svc.Upload(ctx, actor, &FileUploadInput{
		FileName: "notes.txt",
		Size:     int64(len("hello world")),
		Content:  content,
		IsPublic: true,
	})
	if err != nil {
		t.Fatalf("文本文件上传应成功: %v", err)
	}
	var entry struct {
		UploaderID int64
		Uploader   string
		IsPublic   bool
	}
	svc.DB.Table("sys_file").Select("uploader_id", "uploader", "is_public").
		Where("id = ?", result.ID).Scan(&entry)
	if entry.UploaderID != 7 || entry.Uploader != "carol" {
		t.Fatalf("应记录 Actor 为上传人: %+v", entry)
	}
	if !entry.IsPublic {
		t.Fatal("公开标记应写入元数据")
	}
}
