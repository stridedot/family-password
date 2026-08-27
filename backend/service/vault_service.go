package service

import (
	"errors"
	"time"

	"github.com/stridedot/family-password-vault/backend/config"
	"github.com/stridedot/family-password-vault/backend/model"
	"github.com/stridedot/family-password-vault/backend/repository"
	"gorm.io/gorm"
)

// ErrReleased 表示保险库已释放，拒绝主人继续写入。
var ErrReleased = errors.New("vault already released")

// VaultService 承载死亡开关的全部业务逻辑（状态机 + 默认值 + 更新规则），
// 与存储（repository）、HTTP（api）解耦。
type VaultService struct {
	repo *repository.VaultRepository
	cfg  *config.Config
}

func NewVaultService(repo *repository.VaultRepository, cfg *config.Config) *VaultService {
	return &VaultService{repo: repo, cfg: cfg}
}

// Put 创建 / 更新保险库（业务层 upsert）。
// - 新建：填默认值（trigger_status / silence_ms / grace_ms 取配置），打首次心跳。
// - 更新：只改内容字段，保留心跳与触发状态（心跳只由 Heartbeat 改）；
//   silence_ms / grace_ms 只在创建时定一次，之后永不随 .env 变化。
// - 已 released 的库：拒绝主人继续写入（受益人可取走，主人不应再改）。
func (s *VaultService) Put(v *model.Vault) error {
	if v.TriggerStatus == "" {
		v.TriggerStatus = "none"
	}

	existing, err := s.repo.Get(v.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 新建：默认值只在创建时定一次
		if v.SilenceMS == 0 {
			v.SilenceMS = s.cfg.DefaultSilence.Milliseconds()
		}
		if v.GraceMS == 0 {
			v.GraceMS = s.cfg.DefaultGrace.Milliseconds()
		}
		v.CreatedAt = time.Now()
		v.UpdatedAt = time.Now()
		if v.HeartbeatAt == 0 {
			v.HeartbeatAt = model.NowMs()
		}
		return s.repo.Create(v)
	}
	if err != nil {
		return err
	}
	// 已释放：拒绝写入（防止已释放的库被主人覆盖密文 / 重新盖章阈值）
	if existing.TriggerStatus == "released" {
		return ErrReleased
	}
	// 更新：只改内容字段（不覆盖心跳/触发状态/阈值），见 repo.UpdateContent
	return s.repo.UpdateContent(v.ID, v)
}

// Heartbeat 主人报到：刷新心跳 + 取消任何进行中的释放（回到 none）。
func (s *VaultService) Heartbeat(id string) (*model.Vault, error) {
	v, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}
	v.HeartbeatAt = model.NowMs()
	v.TriggerStatus = "none"
	v.GraceEndsAt = 0
	v.ReminderSent = false
	v.GraceReminderSent = false
	v.UpdatedAt = time.Now()
	if err := s.repo.Save(v); err != nil {
		return nil, err
	}
	return v, nil
}

// Evaluate 推进死亡开关状态机（在服务端判定，不依赖客户端）。
// 返回是否发生状态变化，便于调用方决定是否落盘。
func (s *VaultService) Evaluate(v *model.Vault, now int64) bool {
	if v.TriggerStatus == "released" {
		return false
	}
	if now-v.HeartbeatAt >= v.SilenceMS {
		if v.TriggerStatus != "grace" {
			// 首次判定静默超时 → 进入宽限期（主人可反悔）
			v.TriggerStatus = "grace"
			v.GraceEndsAt = now + v.GraceMS
			return true
		}
		if now >= v.GraceEndsAt {
			// 宽限期已过且无人反悔 → 释放
			v.TriggerStatus = "released"
			return true
		}
	}
	return false
}

func (s *VaultService) Get(id string) (*model.Vault, error) { return s.repo.Get(id) }
func (s *VaultService) List() ([]model.Vault, error)         { return s.repo.List() }

// SaveState 整行落盘（不经过 Put 的"只改内容字段"规则）。
// 供 scheduler 持久化状态机推进结果——Evaluate 改动的 TriggerStatus/GraceEndsAt 必须整行写回。
func (s *VaultService) SaveState(v *model.Vault) error {
	return s.repo.Save(v)
}

// Touch 读取并实时推进状态机，变化即落盘。供 GET /:id 与 cron 复用。
func (s *VaultService) Touch(id string) (*model.Vault, bool, error) {
	v, err := s.repo.Get(id)
	if err != nil {
		return nil, false, err
	}
	changed := s.Evaluate(v, time.Now().UnixMilli())
	if changed {
		if err := s.repo.Save(v); err != nil {
			return nil, false, err
		}
	}
	return v, changed, nil
}
