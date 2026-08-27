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

// loadDotEnv 用 godotenv 解析当前目录的 .env 并注入环境变量。
// 先去除 UTF-8 BOM（Windows 编辑器常见）；缺失 .env 时静默跳过。
// 已存在的环境变量不会被覆盖（.env 仅作默认值来源）。
func loadDotEnv() {
	data, err := os.ReadFile(".env")
	if err != nil {
		return
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	env, err := godotenv.Unmarshal(string(data))
	if err != nil {
		return
	}
	for k, v := range env {
		if os.Getenv(k) == "" {
			_ = os.Setenv(k, v)
		}
	}
}
