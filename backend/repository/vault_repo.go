package repository

import (
	"time"

	"github.com/stridedot/family-password-vault/backend/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type VaultRepository struct {
	db *gorm.DB
}

func NewVaultRepository(dbUrl string) (*VaultRepository, error) {
	db, err := gorm.Open(postgres.Open(dbUrl), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.Vault{}); err != nil {
		return nil, err
	}
	return &VaultRepository{db: db}, nil
}

func (r *VaultRepository) Get(id string) (*model.Vault, error) {
	var v model.Vault
	if err := r.db.First(&v, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

// Create 纯持久化：插入新保险库（业务默认值由 service 层填好）。
func (r *VaultRepository) Create(v *model.Vault) error {
	return r.db.Create(v).Error
}

// Save 纯持久化：整行更新（业务规则由 service 层决定改哪些字段）。
func (r *VaultRepository) Save(v *model.Vault) error {
	return r.db.Save(v).Error
}

// UpdateContent 只更新内容字段（targeted update）。
// 不触碰 heartbeat_at / trigger_status / grace_ends_at / silence_ms / grace_ms / created_at，
// 因此并发的心跳 / 状态机写入，以及创建时定下的阈值，都不会被覆盖。
func (r *VaultRepository) UpdateContent(id string, v *model.Vault) error {
	updates := map[string]interface{}{
		"salt":        v.Salt,
		"vault":       v.Vault,
		"beneficiary": v.Beneficiary,
		"updated_at":  time.Now(),
	}
	// email / beneficiary_email 仅当非空时更新，避免更新密文时把已有邮箱清空
	if v.Email != "" {
		updates["email"] = v.Email
	}
	if v.BeneficiaryEmail != "" {
		updates["beneficiary_email"] = v.BeneficiaryEmail
	}
	return r.db.Model(&model.Vault{}).Where("id = ?", id).Updates(updates).Error
}

func (r *VaultRepository) List() ([]model.Vault, error) {
	var vs []model.Vault
	if err := r.db.Find(&vs).Error; err != nil {
		return nil, err
	}
	return vs, nil
}
