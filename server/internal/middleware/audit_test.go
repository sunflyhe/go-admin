package middleware

import "testing"

func TestMaskJSON(t *testing.T) {
	in := `{"username":"alice","password":"plain-secret","nested":{"token":"abc","phone":"13800000000"},"ok":1}`
	out := MaskJSON(in)
	if out == "" {
		t.Fatal("输出为空")
	}
	for _, leaked := range []string{"plain-secret", "abc", "13800000000"} {
		if contains(out, leaked) {
			t.Fatalf("敏感信息泄露: %s 出现在 %s", leaked, out)
		}
	}
	if !contains(out, `"password":"***"`) {
		t.Fatalf("password 未被脱敏: %s", out)
	}
	if !contains(out, `"nested":{"token":"***"`) && !contains(out, `"token":"***"`) {
		t.Fatalf("嵌套 token 未被脱敏: %s", out)
	}
	// 非敏感字段保留
	if !contains(out, `"username":"alice"`) {
		t.Fatalf("非敏感字段不应被修改: %s", out)
	}
}

func TestMaskNonJSON(t *testing.T) {
	// 无法解析为 JSON 的请求体不落库,返回占位
	out := MaskJSON("username=admin&password=123")
	if contains(out, "123") {
		t.Fatalf("非 JSON 请求体不应原样保留: %s", out)
	}
	if MaskJSON("") != "" {
		t.Fatal("空体应返回空")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
