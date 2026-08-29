// Package config 负责配置加载:config.yaml 文件 + 环境变量覆盖,并在启动时校验必填项。
// 环境变量命名规则:ADMIN_<节>_<键>,全大写下划线,例如 ADMIN_MYSQL_DSN、ADMIN_JWT_SECRET。
package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Server struct {
	Addr           string   `yaml:"addr"`
	TrustedProxies []string `yaml:"trustedProxies"`
	// Gin 运行模式:debug(开发,带调试输出)/ release(生产,静默)/ test(测试,静默)。
	// 仅影响 Gin 自身日志,不影响业务逻辑;生产环境必须用 release。
	Mode   string `yaml:"mode"`   // debug | release | test
	WebDir string `yaml:"webDir"` // 前端静态文件目录(可选,设置后由后端托管 web/dist)
}

type MySQL struct {
	DSN                string `yaml:"dsn"`                // 必填:MySQL 连接串,可被 ADMIN_MYSQL_DSN 覆盖
	MaxOpenConns       int    `yaml:"maxOpenConns"`       // 最大打开连接数,默认 50
	MaxIdleConns       int    `yaml:"maxIdleConns"`       // 最大空闲连接数,默认 10
	ConnMaxLifetimeSec int    `yaml:"connMaxLifetimeSec"` // 连接最大存活秒数,默认 3600
	SlowThresholdMs    int    `yaml:"slowThresholdMs"`    // GORM 慢查询阈值毫秒,默认 500
}

type JWT struct {
	Secret           string `yaml:"secret"`           // 必填:签名密钥,≥16 位,可被 ADMIN_JWT_SECRET 覆盖
	AccessTTLMinutes int    `yaml:"accessTTLMinutes"` // access token 有效期分钟,默认 30
	RefreshTTLHours  int    `yaml:"refreshTTLHours"`  // refresh token 有效期小时,默认 168(7 天)
	Issuer           string `yaml:"issuer"`           // JWT iss 声明,默认 go-admin
}

type Log struct {
	Level string `yaml:"level"` // debug | info | warn | error
}

type Upload struct {
	Dir       string `yaml:"dir"`       // 上传存储根目录,默认 ./data/uploads
	MaxSizeMB int    `yaml:"maxSizeMB"` // 单文件大小上限 MB,默认 20
	PublicURL string `yaml:"publicURL"` // 公开文件访问前缀,须以 / 开头,默认 /files
}

type Audit struct {
	RetentionDays int `yaml:"retentionDays"` // 操作日志保存天数,0 表示永久
}

type Config struct {
	Server Server `yaml:"server"`
	MySQL  MySQL  `yaml:"mysql"`
	JWT    JWT    `yaml:"jwt"`
	Log    Log    `yaml:"log"`
	Upload Upload `yaml:"upload"`
	Audit  Audit  `yaml:"audit"`
}

// Load 读取配置文件并用环境变量覆盖,随后校验必填项。
func Load(path string) (*Config, error) {
	cfg := &Config{}
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("读取配置文件 %s: %w", path, err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("解析配置文件 %s: %w", path, err)
		}
	}
	applyEnv(cfg)
	setDefaults(cfg)
	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// envBindings 显式声明环境变量覆盖点,便于审计与文档同步。
func applyEnv(cfg *Config) {
	str := func(env string, dst *string) {
		if v, ok := os.LookupEnv(env); ok && v != "" {
			*dst = v
		}
	}
	intVal := func(env string, dst *int) {
		if v, ok := os.LookupEnv(env); ok && v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				*dst = n
			}
		}
	}
	str("ADMIN_SERVER_ADDR", &cfg.Server.Addr)
	str("ADMIN_SERVER_MODE", &cfg.Server.Mode)
	str("ADMIN_SERVER_WEB_DIR", &cfg.Server.WebDir)
	if v, ok := os.LookupEnv("ADMIN_SERVER_TRUSTED_PROXIES"); ok {
		cfg.Server.TrustedProxies = splitCSV(v)
	}
	str("ADMIN_MYSQL_DSN", &cfg.MySQL.DSN)
	intVal("ADMIN_MYSQL_MAX_OPEN_CONNS", &cfg.MySQL.MaxOpenConns)
	str("ADMIN_JWT_SECRET", &cfg.JWT.Secret)
	intVal("ADMIN_JWT_ACCESS_TTL_MINUTES", &cfg.JWT.AccessTTLMinutes)
	intVal("ADMIN_JWT_REFRESH_TTL_HOURS", &cfg.JWT.RefreshTTLHours)
	str("ADMIN_LOG_LEVEL", &cfg.Log.Level)
	str("ADMIN_UPLOAD_DIR", &cfg.Upload.Dir)
	intVal("ADMIN_UPLOAD_MAX_SIZE_MB", &cfg.Upload.MaxSizeMB)
	intVal("ADMIN_AUDIT_RETENTION_DAYS", &cfg.Audit.RetentionDays)
}

func splitCSV(value string) []string {
	var values []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
}

func setDefaults(cfg *Config) {
	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":8080"
	}
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "release"
	}
	if cfg.MySQL.MaxOpenConns <= 0 {
		cfg.MySQL.MaxOpenConns = 50
	}
	if cfg.MySQL.MaxIdleConns <= 0 {
		cfg.MySQL.MaxIdleConns = 10
	}
	if cfg.MySQL.ConnMaxLifetimeSec <= 0 {
		cfg.MySQL.ConnMaxLifetimeSec = 3600
	}
	if cfg.MySQL.SlowThresholdMs <= 0 {
		cfg.MySQL.SlowThresholdMs = 500
	}
	if cfg.JWT.AccessTTLMinutes <= 0 {
		cfg.JWT.AccessTTLMinutes = 30
	}
	if cfg.JWT.RefreshTTLHours <= 0 {
		cfg.JWT.RefreshTTLHours = 168 // 7 天
	}
	if cfg.JWT.Issuer == "" {
		cfg.JWT.Issuer = "go-admin"
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Upload.Dir == "" {
		cfg.Upload.Dir = "./data/uploads"
	}
	if cfg.Upload.MaxSizeMB <= 0 {
		cfg.Upload.MaxSizeMB = 20
	}
	if cfg.Upload.PublicURL == "" {
		cfg.Upload.PublicURL = "/files"
	}
	if cfg.Audit.RetentionDays < 0 {
		cfg.Audit.RetentionDays = 0
	}
}

// validate 启动时强制校验必填项与安全下限。
func validate(cfg *Config) error {
	var missing []string
	if cfg.MySQL.DSN == "" {
		missing = append(missing, "mysql.dsn(或环境变量 ADMIN_MYSQL_DSN)")
	}
	if cfg.JWT.Secret == "" {
		missing = append(missing, "jwt.secret(或环境变量 ADMIN_JWT_SECRET)")
	}
	if len(missing) > 0 {
		return fmt.Errorf("缺少必填配置: %s", strings.Join(missing, "; "))
	}
	if len(cfg.JWT.Secret) < 16 {
		return fmt.Errorf("jwt.secret 长度不得少于 16 位")
	}
	switch cfg.Server.Mode {
	case "debug", "release", "test":
	default:
		return fmt.Errorf("server.mode 仅支持 debug/release/test,当前: %q", cfg.Server.Mode)
	}
	for _, proxy := range cfg.Server.TrustedProxies {
		if net.ParseIP(proxy) == nil {
			if _, _, err := net.ParseCIDR(proxy); err != nil {
				return fmt.Errorf("server.trustedProxies 包含无效 IP 或 CIDR: %q", proxy)
			}
		}
	}
	return nil
}
