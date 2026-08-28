package config

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config 集中存放可环境变量覆盖的运行参数。
type Config struct {
	Port           string
	DBUrl          string
	DefaultSilence time.Duration // 静默多久触发释放
	DefaultGrace   time.Duration // 反悔窗口长度
	CronSpec       string        // cron 表达式（含秒，6 段）

	// 邮件通知（可选；未配 SMTP 时仅控制台打印，方便本地跑通）
	BaseURL      string        // 后端对外地址，用于邮件里的"确认存活"链接；缺省 http://localhost:Port
	AppURL       string        // 前端应用对外地址，用于邮件里给受益人的"取用链接"；缺省同 BaseURL
	ReminderLead time.Duration // 静默阈值剩余多少天开始发预警邮件
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPass     string
	FromAddr     string
	FromName     string

	// CORS：允许跨域来源。默认 *（任何来源，本地开发方便）；
	// 生产建议填前端域名（如 https://family-password.vercel.app），限制其他网站调用后端。
	AllowOrigin string
}

func Load() *Config {
	loadDotEnv()
	silenceDays := getEnvInt("SILENCE_DAYS", 30)
	graceHours := getEnvInt("GRACE_HOURS", 336) // 默认 14 天宽限期（=336h）
	reminderDays := getEnvInt("REMINDER_DAYS", 7)
	smtpPort := getEnvInt("SMTP_PORT", 0)

	return &Config{
		Port:           getEnv("PORT", "8080"),
		DBUrl:          getEnv("DATABASE_URL_POOLED", ""),
		DefaultSilence: time.Duration(silenceDays) * 24 * time.Hour,
		DefaultGrace:   time.Duration(graceHours) * time.Hour,
		CronSpec:       getEnv("CRON_SPEC", "0 0 * * * *"),

		BaseURL:      getEnv("BASE_URL", ""),
		AppURL:       getEnv("APP_URL", ""),
		ReminderLead: time.Duration(reminderDays) * 24 * time.Hour,
		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     smtpPort,
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPass:     getEnv("SMTP_PASS", ""),
		FromAddr:     getEnv("MAIL_FROM", ""),
		FromName:     getEnv("MAIL_FROM_NAME", "家庭密码"),
		AllowOrigin:  getEnv("ALLOW_ORIGIN", "*"),
	}
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// getEnvInt 读取整型环境变量；为空或非法（非数字、负数）时回退默认值。
// 注意：显式设为 0 是允许的（用于本地"立即触发"测试），仅负数判无效。
func getEnvInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n < 0 {
		return def
	}
	return n
}

// loadDotEnv 用 godotenv 解析环境文件并注入环境变量。
// 支持多环境：APP_ENV=production 时优先读 .env.production，否则读 .env。
// 顺序：更具体的环境文件优先；已有环境变量（如 Render dashboard 注入的）永远优先，文件不覆盖。
// 先去除 UTF-8 BOM（Windows 编辑器常见）；文件缺失时静默跳过。
//
// 用法：
//   - 本地开发：go run main.go（读 .env，连本地库）
//   - 本地模拟生产：APP_ENV=production go run main.go（读 .env.production，连生产库）
//   - 线上 Render：不打包 .env 文件，直接用 dashboard 环境变量，本函数无副作用
func loadDotEnv() {
	candidates := []string{}
	if env := os.Getenv("APP_ENV"); env != "" {
		candidates = append(candidates, ".env."+env)
	}
	candidates = append(candidates, ".env")

	for _, f := range candidates {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
		parsed, err := godotenv.Unmarshal(string(data))
		if err != nil {
			continue
		}
		for k, v := range parsed {
			if os.Getenv(k) == "" {
				_ = os.Setenv(k, v)
			}
		}
	}
}
