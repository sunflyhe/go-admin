// 文件服务测试:不构造 Gin 上下文,以文件流直接调用,
// 验证扩展名白名单、真实 MIME 嗅探校验与上传人记录不因脱离 HTTP 而回退。
package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/model"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/errs"
	"github.com/hesunfly/hesunfly-admin-go/server/pkg/page"
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

// historicalWhitelist 是上传白名单的字面量集合。
// 用它钉死不变量:调整类型标签不能顺手放宽上传,新增扩展名必须显式归类。
var historicalWhitelist = []string{
	".jpg", ".jpeg", ".png", ".gif", ".webp",
	".pdf", ".txt", ".csv", ".xlsx", ".xls", ".docx", ".zip",
}

func TestFileExtKindIsTheUploadWhitelist(t *testing.T) {
	if len(fileExtKind) != len(historicalWhitelist) {
		t.Fatalf("上传白名单条目数变了: %d vs %d", len(fileExtKind), len(historicalWhitelist))
	}
	for _, ext := range historicalWhitelist {
		kind, ok := fileExtKind[ext]
		if !ok {
			t.Fatalf("扩展名 %s 被从上传白名单移除", ext)
		}
		switch kind {
		case kindImage, kindDocument, kindArchive:
		default:
			t.Fatalf("扩展名 %s 必须归入某个具体类别,实际 %q", ext, kind)
		}
	}

	// 三个归类互斥且并集覆盖全部白名单键
	seen := map[string]fileKind{}
	for _, kind := range []fileKind{kindImage, kindDocument, kindArchive} {
		exts := extsInKind(kind)
		if len(exts) == 0 {
			t.Fatalf("归类 %s 没有任何扩展名", kind)
		}
		if !sort.StringsAreSorted(exts) {
			t.Fatalf("归类 %s 的扩展名未排序,SQL 文本会不稳定: %v", kind, exts)
		}
		for _, ext := range exts {
			if prev, dup := seen[ext]; dup {
				t.Fatalf("扩展名 %s 同时属于 %s 与 %s", ext, prev, kind)
			}
			seen[ext] = kind
		}
	}
	if len(seen) != len(fileExtKind) {
		t.Fatalf("归类并集 %d 与白名单 %d 不一致", len(seen), len(fileExtKind))
	}
	if len(knownExts()) != len(fileExtKind) {
		t.Fatalf("knownExts 应覆盖全部白名单键: %v", knownExts())
	}
}

func TestNormalizeCategory(t *testing.T) {
	for raw, want := range map[string]FileCategory{
		"":      FileCategoryAll,
		"all":   FileCategoryAll,
		" ALL ": FileCategoryAll,
		"Image": FileCategoryImage,
		" file": FileCategoryFile,
		"video": FileCategoryVideo,
	} {
		got, err := normalizeCategory(raw)
		if err != nil {
			t.Fatalf("normalizeCategory(%q) 报错: %v", raw, err)
		}
		if got != want {
			t.Fatalf("normalizeCategory(%q) = %q,期望 %q", raw, got, want)
		}
	}
	// document/archive/other 是旧的上传归类,不再是对外标签
	for _, raw := range []string{"document", "archive", "other", "audio"} {
		if _, err := normalizeCategory(raw); err == nil {
			t.Fatalf("标签 %q 应报错,而不是静默退化成全部", raw)
		}
	}
}

// fileSeed 一行文件元数据。直接写库而不是走 Upload,才能构造出 .mp4、.exe、
// 空扩展名这些上传白名单不允许、但列表标签必须正确归类的历史数据。
type fileSeed struct {
	ext     string
	mime    string
	groupID int64
}

// seedFiles 写入元数据行,返回与入参同序的自增 id。
func seedFiles(t *testing.T, svc *FileService, seeds ...fileSeed) []int64 {
	t.Helper()
	ids := make([]int64, 0, len(seeds))
	for i, seed := range seeds {
		ext := seed.ext
		entry := model.SysFile{
			OriginName: fmt.Sprintf("f%d%s", i, ext),
			StorePath:  fmt.Sprintf("2026/01/01/f%d%s", i, ext),
			Size:       10,
			MIME:       seed.mime,
			Ext:        ext,
			GroupID:    seed.groupID,
		}
		if err := svc.DB.Create(&entry).Error; err != nil {
			t.Fatal(err)
		}
		ids = append(ids, entry.ID)
	}
	return ids
}

// listExtsFor 返回按分类筛选后的扩展名集合(升序),便于集合断言。
func listExtsFor(t *testing.T, svc *FileService, input *FileListInput) ([]string, int64) {
	t.Helper()
	res, err := svc.List(context.Background(), input)
	if err != nil {
		t.Fatalf("列表查询失败: %v", err)
	}
	rows, ok := res.List.([]model.SysFile)
	if !ok {
		t.Fatalf("列表返回类型变了: %T", res.List)
	}
	exts := make([]string, 0, len(rows))
	for _, r := range rows {
		exts = append(exts, r.Ext)
	}
	sort.Strings(exts)
	return exts, res.Total
}

func assertExts(t *testing.T, got, want []string, label string) {
	t.Helper()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("%s 命中集合不符: got=%v want=%v", label, got, want)
	}
}

func listBy(category FileCategory) *FileListInput {
	return &FileListInput{Query: page.Query{Page: 1, PageSize: 50}, Category: category}
}

// mixedSeed 覆盖四档标签的全部边界:
// .JPG 证明 LOWER 生效(SQLite 字符串比较大小写敏感,不降小写就会漏);
// .mp4 是白名单外的历史数据,只能靠 mime 落进"视频";
// .zip 归"文件"(旧的 archive 一档已取消);空扩展名与 .exe 也必须被"文件"兜住。
var mixedSeed = []fileSeed{
	{ext: ".png", mime: "image/png"},
	{ext: ".JPG", mime: "image/jpeg"},
	{ext: ".mp4", mime: "video/mp4"},
	{ext: ".pdf", mime: "application/pdf"},
	{ext: ".zip", mime: "application/zip"},
	{ext: ".exe", mime: "application/octet-stream"},
	{ext: "", mime: "application/octet-stream"},
}

func TestListCategoryFilters(t *testing.T) {
	svc, _ := newFileService(t)
	seedFiles(t, svc, mixedSeed...)

	image, imageTotal := listExtsFor(t, svc, listBy(FileCategoryImage))
	assertExts(t, image, []string{".JPG", ".png"}, "image")
	if imageTotal != 2 {
		t.Fatalf("image total 应为 2: %d", imageTotal)
	}

	video, videoTotal := listExtsFor(t, svc, listBy(FileCategoryVideo))
	assertExts(t, video, []string{".mp4"}, "video")

	// "文件"是图片与视频的补集:文档、压缩包、白名单外的 .exe 与历史空扩展名都在这
	file, fileTotal := listExtsFor(t, svc, listBy(FileCategoryFile))
	assertExts(t, file, []string{"", ".exe", ".pdf", ".zip"}, "file")

	all, allTotal := listExtsFor(t, svc, listBy(FileCategoryAll))
	assertExts(t, all, []string{"", ".JPG", ".exe", ".mp4", ".pdf", ".png", ".zip"}, "all")

	// 三档互斥:任何一行都不能被两个标签同时命中
	for _, pair := range [][2][]string{{image, video}, {image, file}, {video, file}} {
		for _, x := range pair[0] {
			for _, y := range pair[1] {
				if x == y {
					t.Fatalf("扩展名 %q 被两个标签同时命中", x)
				}
			}
		}
	}
	// 三档并集等于全部:不允许出现"哪一档都进不去"的孤儿文件
	if imageTotal+videoTotal+fileTotal != allTotal {
		t.Fatalf("三档合计 %d 与总数 %d 不一致", imageTotal+videoTotal+fileTotal, allTotal)
	}
}

func TestListCategoryCombinesWithOriginName(t *testing.T) {
	svc, _ := newFileService(t)
	seedFiles(t, svc, mixedSeed...)

	// category 与 originName 是 AND 语义:关键词命中 7 行,但图片只有 2 行
	input := listBy(FileCategoryImage)
	input.OriginName = "f"
	got, total := listExtsFor(t, svc, input)
	if total != 2 || len(got) != 2 {
		t.Fatalf("AND 语义下应仍为 2 条: total=%d got=%v", total, got)
	}

	// 关键词收窄到 1 行时,分类不匹配就应为空(f3 是 pdf)
	input.OriginName = "f3"
	got, total = listExtsFor(t, svc, input)
	if total != 0 || len(got) != 0 {
		t.Fatalf("f3 是 pdf,图片筛选应为空: total=%d got=%v", total, got)
	}
}

func TestListRejectsUnknownCategory(t *testing.T) {
	svc, _ := newFileService(t)
	seedFiles(t, svc, fileSeed{ext: ".png", mime: "image/png"})

	_, err := svc.List(context.Background(), listBy(FileCategory("document")))
	var appErr *errs.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("未知分类应返回参数错误,实际 %v", err)
	}
	if appErr.HTTP != http.StatusBadRequest {
		t.Fatalf("未知分类应返回 400: %d", appErr.HTTP)
	}
}

func TestListGroupFilter(t *testing.T) {
	svc, _ := newFileService(t)
	seedFiles(t, svc,
		fileSeed{ext: ".png", mime: "image/png", groupID: 1},
		fileSeed{ext: ".pdf", mime: "application/pdf", groupID: 1},
		fileSeed{ext: ".zip", mime: "application/zip", groupID: 2},
		fileSeed{ext: ".txt", mime: "text/plain"}, // group_id=0 即"未分组"
	)

	list := func(groupID *int64) []string {
		input := listBy(FileCategoryAll)
		input.GroupID = groupID
		got, _ := listExtsFor(t, svc, input)
		return got
	}
	one, zero := int64(1), int64(0)

	assertExts(t, list(nil), []string{".pdf", ".png", ".txt", ".zip"}, "全部分组")
	assertExts(t, list(&one), []string{".pdf", ".png"}, "分组 1")
	assertExts(t, list(&zero), []string{".txt"}, "未分组")

	// 分组与标签是 AND:分组 1 里的图片只有那张 png
	input := listBy(FileCategoryImage)
	input.GroupID = &one
	got, _ := listExtsFor(t, svc, input)
	assertExts(t, got, []string{".png"}, "分组 1 的图片")
}

func groupIDOf(t *testing.T, svc *FileService, id int64) int64 {
	t.Helper()
	var row model.SysFile
	if err := svc.DB.First(&row, id).Error; err != nil {
		t.Fatal(err)
	}
	return row.GroupID
}

// assertHTTPStatus 断言错误是业务错误且 HTTP 状态码符合预期:
// 只断言"报错了"会把 500 也放行,掩盖真正的实现缺陷。
func assertHTTPStatus(t *testing.T, err error, code int, label string) {
	t.Helper()
	var appErr *errs.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("%s 应返回业务错误,实际 %v", label, err)
	}
	if appErr.HTTP != code {
		t.Fatalf("%s 应返回 %d,实际 %d: %v", label, code, appErr.HTTP, err)
	}
}

func TestMoveChangesOnlySelectedRows(t *testing.T) {
	svc, ctx := newFileService(t)
	ids := seedFiles(t, svc,
		fileSeed{ext: ".png", mime: "image/png"},
		fileSeed{ext: ".pdf", mime: "application/pdf"},
	)
	if err := svc.DB.Create(&model.SysFileGroup{Name: "素材"}).Error; err != nil {
		t.Fatal(err)
	}
	var groupID int64
	if err := svc.DB.Model(&model.SysFileGroup{}).Select("id").Row().Scan(&groupID); err != nil {
		t.Fatal(err)
	}

	if err := svc.Move(ctx, []int64{ids[0]}, groupID); err != nil {
		t.Fatalf("移动失败: %v", err)
	}
	if got := groupIDOf(t, svc, ids[0]); got != groupID {
		t.Fatalf("选中行应进入目标分组: %d vs %d", got, groupID)
	}
	if got := groupIDOf(t, svc, ids[1]); got != 0 {
		t.Fatalf("未选中的行不应被动过: %d", got)
	}

	// 重复移到同一分组必须成功:MySQL 在"更新成相同值"时 RowsAffected=0,
	// 靠它判定存在性会把幂等操作误报成失败。
	if err := svc.Move(ctx, []int64{ids[0]}, groupID); err != nil {
		t.Fatalf("重复移动应成功: %v", err)
	}

	if err := svc.Move(ctx, []int64{ids[0]}, 0); err != nil {
		t.Fatalf("移回未分组失败: %v", err)
	}
	if got := groupIDOf(t, svc, ids[0]); got != 0 {
		t.Fatalf("应回到未分组: %d", got)
	}
}

func TestMoveRejects(t *testing.T) {
	svc, ctx := newFileService(t)
	ids := seedFiles(t, svc, fileSeed{ext: ".png", mime: "image/png"})

	assertHTTPStatus(t, svc.Move(ctx, nil, 1), http.StatusBadRequest, "空集合")
	assertHTTPStatus(t, svc.Move(ctx, []int64{0, -1}, 1), http.StatusBadRequest, "全非法 id")
	assertHTTPStatus(t, svc.Move(ctx, ids, 999), http.StatusNotFound, "目标分组不存在")
	assertHTTPStatus(t, svc.Move(ctx, []int64{ids[0], 999}, 0), http.StatusNotFound, "含不存在的文件 id")
	// 存在性校验先于更新:有一个 id 不存在时整批不得有变更
	if got := groupIDOf(t, svc, ids[0]); got != 0 {
		t.Fatalf("校验失败时不应有任何变更: %d", got)
	}
}

func TestBatchDeleteRemovesRowsAndBlobs(t *testing.T) {
	svc, ctx := newFileService(t)
	actor := Actor{ID: 1, Username: "admin", IsSuper: true}
	content := append(append([]byte{}, pngMagic...), bytes.Repeat([]byte{0}, 32)...)

	var ids []int64
	var paths []string
	for i := 0; i < 2; i++ {
		res, err := svc.Upload(ctx, actor, &FileUploadInput{
			FileName: fmt.Sprintf("b%d.png", i), Size: int64(len(content)),
			Content: bytes.NewReader(content),
		})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, res.ID)
		paths = append(paths, res.StorePath)
	}
	// 只有元数据、磁盘上没有对应文件的行也要能删:Storage.Delete 的 ErrNotExist 被容忍
	ids = append(ids, seedFiles(t, svc, fileSeed{ext: ".txt", mime: "text/plain"})...)

	if err := svc.BatchDelete(ctx, ids); err != nil {
		t.Fatalf("批量删除失败: %v", err)
	}
	var left int64
	if err := svc.DB.Model(&model.SysFile{}).Count(&left).Error; err != nil {
		t.Fatal(err)
	}
	if left != 0 {
		t.Fatalf("应删掉全部 %d 行,剩余 %d", len(ids), left)
	}
	for _, p := range paths {
		if f, err := svc.Storage.Open(p); err == nil {
			_ = f.Close()
			t.Fatalf("磁盘文件应被删除: %s", p)
		}
	}
}

func TestBatchDeleteRejects(t *testing.T) {
	svc, ctx := newFileService(t)
	ids := seedFiles(t, svc,
		fileSeed{ext: ".png", mime: "image/png"},
		fileSeed{ext: ".pdf", mime: "application/pdf"},
	)

	err := svc.BatchDelete(ctx, []int64{ids[0], 999})
	var appErr *errs.AppError
	if !errors.As(err, &appErr) || appErr.HTTP != http.StatusNotFound {
		t.Fatalf("含不存在 id 应整批失败: %v", err)
	}
	var left int64
	if err := svc.DB.Model(&model.SysFile{}).Count(&left).Error; err != nil {
		t.Fatal(err)
	}
	if left != 2 {
		t.Fatalf("整批失败时不应删掉任何行: %d", left)
	}
	if err := svc.BatchDelete(ctx, nil); err == nil {
		t.Fatal("空集合应报错,而不是静默返回成功")
	}
}

func TestNormalizeIDs(t *testing.T) {
	got, err := normalizeIDs([]int64{3, 1, 3, 0, -2, 1})
	if err != nil {
		t.Fatalf("应通过: %v", err)
	}
	if len(got) != 2 || got[0] != 3 || got[1] != 1 {
		t.Fatalf("应去重、剔除非法 id 并保持原顺序: %v", got)
	}

	many := make([]int64, 0, maxBatchIDs+1)
	for i := int64(1); i <= maxBatchIDs+1; i++ {
		many = append(many, i)
	}
	if _, err := normalizeIDs(many); err == nil {
		t.Fatal("超过单次批量上限应报错")
	}
}

func TestUploadPersistsGroupID(t *testing.T) {
	svc, ctx := newFileService(t)
	actor := Actor{ID: 1, Username: "admin", IsSuper: true}
	content := append(append([]byte{}, pngMagic...), bytes.Repeat([]byte{0}, 32)...)

	res, err := svc.Upload(ctx, actor, &FileUploadInput{
		FileName: "in-group.png", Size: int64(len(content)),
		Content: bytes.NewReader(content), GroupID: 5,
	})
	if err != nil {
		t.Fatalf("上传失败: %v", err)
	}
	if got := groupIDOf(t, svc, res.ID); got != 5 {
		t.Fatalf("上传应落到指定分组: %d", got)
	}
}
