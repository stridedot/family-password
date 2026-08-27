package model

import "time"

// Vault 是服务端唯一持久化的内容：只有密文 + 心跳 + 触发状态机。
// 主密钥 K 与明文永远不在服务端——这是零知识产品的根基。
type Vault struct {
	ID                string    `gorm:"primaryKey;type:uuid" json:"id"`
	Salt              string    `gorm:"column:salt;not null;default:''" json:"salt"`                                  // base64，客户端派生密钥用
	Vault             string    `gorm:"column:vault;not null;default:''" json:"vault"`                                // base64 JSON {iv,ct}，用主密码加密
	Beneficiary       string    `gorm:"column:beneficiary;not null;default:''" json:"beneficiary"`                    // base64 JSON {iv,ct}，用释放密码加密
	Email             string    `gorm:"column:email; not null;default:''" json:"email"`                               // 主人通知邮箱（仅通知用途，非账户/登录）
	BeneficiaryEmail  string    `gorm:"column:beneficiary_email; not null;default:''" json:"beneficiary_email"`       // 受益人邮箱：释放时自动通知TA来取用（仅元数据/PII，不存密文，Get 响应脱敏不返回）
	HeartbeatAt       int64     `gorm:"column:heartbeat_at;not null;default:0" json:"heartbeat_at"`                   // 上次"我还活着"时间戳（unix ms）
	ReminderSent      bool      `gorm:"column:reminder_sent;not null;default:false" json:"reminder_sent"`             // 本轮静默是否已发过预警邮件（防每小时重复发）
	GraceReminderSent bool      `gorm:"column:grace_reminder_sent;not null;default:false" json:"grace_reminder_sent"` // 宽限期内是否已发过最终提醒（防重复发）
	TriggerStatus     string    `gorm:"column:trigger_status;not null;default:'none'" json:"trigger_status"`          // none | grace | released
	GraceEndsAt       int64     `gorm:"column:grace_ends_at;not null;default:0" json:"grace_ends_at"`                 // 宽限期结束（unix ms）
	SilenceMS         int64     `gorm:"column:silence_ms;not null;default:0" json:"silence_ms"`                       // 静默阈值（默认 30 天）
	GraceMS           int64     `gorm:"column:grace_ms;not null;default:0" json:"grace_ms"`                           // 宽限期长度（默认 14 天）
	CreatedAt         time.Time `gorm:"column:created_at;default:(-)" json:"created_at"`                              // 审计：gorm 自动填充（TIMESTAMPTZ）
	UpdatedAt         time.Time `gorm:"column:updated_at;default:(-)" json:"updated_at"`                              // 审计：gorm 自动填充（TIMESTAMPTZ）
}

func (*Vault) TableName() string { return "vaults" }

func NowMs() int64 { return time.Now().UnixMilli() }
