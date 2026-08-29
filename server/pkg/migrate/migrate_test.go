// 迁移文件自检:命名合法、up/down 成对、版本号连续,且能被 golang-migrate 的 iofs 数据源解析。
// 这些约束只在服务启动执行迁移时才会暴露,放到测试里提前失败。
package migrate

import (
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4/source/iofs"

	migrations "github.com/hesunfly/hesunfly-admin-go/server/migrations"
)

func TestEmbeddedMigrationsArePairedAndContiguous(t *testing.T) {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		t.Fatalf("读取内嵌迁移目录失败: %v", err)
	}

	type pair struct{ up, down bool }
	versions := map[int]*pair{}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		parts := strings.Split(name, ".")
		if len(parts) != 3 || (parts[1] != "up" && parts[1] != "down") {
			t.Fatalf("迁移文件命名不合法(应为 NNNNNN_描述.up|down.sql): %s", name)
		}
		digits := strings.SplitN(parts[0], "_", 2)[0]
		version, err := strconv.Atoi(digits)
		if err != nil || version <= 0 {
			t.Fatalf("迁移版本号无法解析: %s", name)
		}
		if versions[version] == nil {
			versions[version] = &pair{}
		}
		if parts[1] == "up" {
			versions[version].up = true
		} else {
			versions[version].down = true
		}
	}

	if len(versions) == 0 {
		t.Fatal("未找到任何迁移文件")
	}
	nums := make([]int, 0, len(versions))
	for v := range versions {
		nums = append(nums, v)
	}
	sort.Ints(nums)

	for i, v := range nums {
		if v != i+1 {
			t.Fatalf("版本号必须从 1 连续递增,断在 %d(实际序列 %v)", v, nums)
		}
		if !versions[v].up || !versions[v].down {
			t.Fatalf("版本 %d 缺少 %s", v, map[bool]string{true: "up", false: "down"}[versions[v].up])
		}
	}
}

func TestMigrationSourceLoads(t *testing.T) {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		t.Fatalf("iofs 无法解析内嵌迁移: %v", err)
	}
	defer src.Close()

	first, err := src.First()
	if err != nil {
		t.Fatalf("读取首个迁移版本失败: %v", err)
	}
	if first != 1 {
		t.Fatalf("首个迁移版本应为 1: %d", first)
	}
	last := first
	for {
		next, err := src.Next(last)
		if err != nil {
			break
		}
		if next != last+1 {
			t.Fatalf("版本不连续: %d 之后是 %d", last, next)
		}
		last = next
	}
}
