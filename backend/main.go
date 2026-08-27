package main

import (
	"log"

	"github.com/stridedot/family-password-vault/backend/config"
	"github.com/stridedot/family-password-vault/backend/repository"
	"github.com/stridedot/family-password-vault/backend/router"
	"github.com/stridedot/family-password-vault/backend/scheduler"
	"github.com/stridedot/family-password-vault/backend/service"
)

func main() {
	cfg := config.Load()

	repo, err := repository.NewVaultRepository(cfg.DBUrl)
	if err != nil {
		log.Fatalf("init db: %v", err)
	}

	svc := service.NewVaultService(repo, cfg)

	c := scheduler.StartCron(svc, cfg)
	if c != nil {
		defer c.Stop()
	}

	r := router.New(svc)
	addr := ":" + cfg.Port
	log.Printf("家庭密码 后端 → http://localhost%s", addr)
	log.Printf("默认静默=%.0f天，反悔窗口=%.0f小时；cron=%s", cfg.DefaultSilence.Hours()/24, cfg.DefaultGrace.Hours(), cfg.CronSpec)
	log.Fatal(r.Run(addr))
}
