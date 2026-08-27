package notify

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"github.com/stridedot/family-password-vault/backend/config"
)

// Sender 邮件发送抽象。便于演示用 Console，生产用 SMTP。
type Sender interface {
	Send(to, subject, body string) error
}

// ConsoleSender 仅打印邮件内容（无 SMTP 配置时回退，方便本地跑通整条链路）
type ConsoleSender struct{}

func (ConsoleSender) Send(to, subject, body string) error {
	log.Printf("[mail] → %s | %s\n%s", to, subject, body)
	return nil
}

// SMTPSender 通过 SMTP 发送真实邮件
type SMTPSender struct {
	Host, User, Pass, FromAddr, FromName string
	Port                                 int
}

func (s *SMTPSender) Send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	var auth smtp.Auth
	if s.User != "" {
		auth = smtp.PlainAuth("", s.User, s.Pass, s.Host)
	}
	msg := buildMime(s.FromAddr, s.FromName, to, subject, body)
	return smtp.SendMail(addr, auth, s.FromAddr, []string{to}, msg)
}

// buildMime 拼一封最简 text/plain(UTF-8) 邮件
func buildMime(fromAddr, fromName, to, subject, body string) []byte {
	var b strings.Builder
	if fromName != "" {
		_, _ = fmt.Fprintf(&b, "From: %s <%s>\r\n", fromName, fromAddr)
	} else {
		_, _ = fmt.Fprintf(&b, "From: %s\r\n", fromAddr)
	}
	_, _ = fmt.Fprintf(&b, "To: %s\r\n", to)
	_, _ = fmt.Fprintf(&b, "Subject: %s\r\n", subject)
	_, _ = fmt.Fprintf(&b, "MIME-Version: 1.0\r\n")
	_, _ = fmt.Fprintf(&b, "Content-Type: text/plain; charset=UTF-8\r\n")
	_, _ = fmt.Fprintf(&b, "\r\n")
	b.WriteString(body)
	return []byte(b.String())
}

// NewSender 有 SMTP 配置则发真邮件，否则回退控制台打印（演示用）
func NewSender(cfg *config.Config) Sender {
	if cfg.SMTPHost != "" && cfg.SMTPPort > 0 {
		return &SMTPSender{
			Host: cfg.SMTPHost, Port: cfg.SMTPPort, User: cfg.SMTPUser,
			Pass: cfg.SMTPPass, FromAddr: cfg.FromAddr, FromName: cfg.FromName,
		}
	}
	return ConsoleSender{}
}
