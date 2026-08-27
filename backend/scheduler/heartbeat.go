package scheduler

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/stridedot/family-password-vault/backend/config"
	"github.com/stridedot/family-password-vault/backend/notify"
	"github.com/stridedot/family-password-vault/backend/service"
)

// StartCron 启动定时任务：周期性推进所有保险库的死亡开关状态机，
// 并在关键节点给主人发邮件提醒（避免忘记报到被误释放）。
// 即使没人访问，静默超时的保险库也会在反悔窗口过后自动释放。
// spec 为 6 段 cron（含秒），如 "*/30 * * * * *" = 每 30 秒。
func StartCron(svc *service.VaultService, cfg *config.Config) *cron.Cron {
	c := cron.New(cron.WithSeconds())

	sender := notify.NewSender(cfg)
	spec := cfg.CronSpec

	_, err := c.AddFunc(spec, func() {
		vaults, err := svc.List()
		if err != nil {
			log.Printf("[cron] list error: %v", err)
			return
		}
		log.Printf("[cron] tick — 扫描 %d 个保险库", len(vaults))
		now := time.Now().UnixMilli()
		lead := cfg.ReminderLead.Milliseconds()
		// BASE_URL 应为"完整公网地址"（如 https://family-password.onrender.com，含 scheme，可含端口）。
		// 不能无条件拼 :Port：Render 等平台对外只暴露 443，内部 PORT 是别的值，拼了反而坏链。
		base := cfg.BaseURL
		if base == "" {
			base = "http://localhost:" + cfg.Port
		}

		// 受益人取用链接指向前端应用（index.html 的 ?id= 深链），不是后端 API。
		// 后端没有 / 路由也不托管静态页，故 APP_URL 未配置时不能回退到 base（会 404 死链）。
		// released 分支里对此做了"跳过 + 告警"。
		appURL := cfg.AppURL

		for i := range vaults {
			v := &vaults[i]

			// 1) 静默临近预警：进入"剩余 ≤ lead"窗口且本周期未发过 → 发提醒
			if v.TriggerStatus == "none" && v.Email != "" && !v.ReminderSent {
				remaining := v.SilenceMS - (now - v.HeartbeatAt)
				if remaining > 0 && remaining <= lead {
					link := fmt.Sprintf("%s/confirm/%s", base, v.ID)
					days := v.SilenceMS / 86_400_000
					body := fmt.Sprintf("你的家庭密码保险库已 %d 天未报到。\n若你一切安好，请点击下面的链接确认存活，避免被误释放：\n%s", days, link)
					if mailErr := sender.Send(v.Email, "家庭密码：请确认你还活着", body); mailErr != nil {
						log.Printf("[cron] mail error: %v", mailErr)
					} else {
						v.ReminderSent = true
						if saveErr := svc.SaveState(v); saveErr != nil {
							log.Printf("[cron] save error: %v", saveErr)
						}
					}
				}
			}

			// 2) 状态机推进
			if svc.Evaluate(v, now) {
				if saveErr := svc.SaveState(v); saveErr != nil {
					log.Printf("[cron] save error: %v", saveErr)
				}
				switch v.TriggerStatus {
				case "grace":
					// 进入宽限期：最后通牒，点击链接可取消释放
					if v.Email != "" {
						link := fmt.Sprintf("%s/confirm/%s", base, v.ID)
						body := fmt.Sprintf("你的家庭密码保险库即将释放（宽限期结束即不可逆）。\n若你一切安好，立即点击下面的链接取消释放：\n%s", link)
						if mailErr := sender.Send(v.Email, "家庭密码：保险库即将释放", body); mailErr != nil {
							log.Printf("[cron] mail error: %v", mailErr)
						}
					}
				case "released":
					if v.Email != "" {
						body := "你的家庭密码保险库已释放，受益人现在可以取走内容。"
						if mailErr := sender.Send(v.Email, "家庭密码：保险库已释放", body); mailErr != nil {
							log.Printf("[cron] mail error: %v", mailErr)
						}
					}
					// 通知受益人：释放即自动发（仅此一次，released 为终态，Evaluate 不再变化）。
					// APP_URL 未配置时拿不到可用的取用链接 → 跳过并发告警，避免发出 404 死链。
					if v.BeneficiaryEmail != "" {
						if appURL == "" {
							log.Printf("[cron] APP_URL 未配置，跳过受益人 %s 的释放通知（无法生成取用链接）", v.BeneficiaryEmail)
						} else {
							benLink := fmt.Sprintf("%s/?id=%s", strings.TrimRight(appURL, "/"), v.ID)
							body := fmt.Sprintf("有人为你预留了一份「家庭密码」保险库，现已释放可供取用。\n打开下面的链接，输入当时留给你的释放密码即可查看内容：\n%s", benLink)
							if mailErr := sender.Send(v.BeneficiaryEmail, "家庭密码：一份留给你的保险库已可用", body); mailErr != nil {
								log.Printf("[cron] mail error (beneficiary): %v", mailErr)
							} else {
								log.Printf("[cron] 已向受益人 %s 发送释放通知", v.BeneficiaryEmail)
							}
						}
					}
				}
			}

			// 3) 宽限期内最终提醒：释放前 lead 天再催一次
			if v.TriggerStatus == "grace" && v.Email != "" && !v.GraceReminderSent && now >= v.GraceEndsAt-lead {
				link := fmt.Sprintf("%s/confirm/%s", base, v.ID)
				leftDays := (v.GraceEndsAt - now) / 86_400_000
				body := fmt.Sprintf("最后一次提醒：你的家庭密码保险库将在约 %d 天后释放。\n若你一切安好，立即点击下面的链接取消释放：\n%s", leftDays, link)
				if mailErr := sender.Send(v.Email, "家庭密码：释放倒计时，最后确认", body); mailErr != nil {
					log.Printf("[cron] mail error: %v", mailErr)
				} else {
					v.GraceReminderSent = true
					if saveErr := svc.SaveState(v); saveErr != nil {
						log.Printf("[cron] save error: %v", saveErr)
					}
				}
			}
		}
	})
	if err != nil {
		log.Printf("[cron] add func error: %v", err)
		return nil
	}
	c.Start()
	log.Println("[cron] Scheduler started (with email reminders)")
	return c
}
