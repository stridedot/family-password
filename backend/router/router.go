package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stridedot/family-password-vault/backend/api"
	"github.com/stridedot/family-password-vault/backend/config"
	"github.com/stridedot/family-password-vault/backend/service"
)

// cors 允许前端跨域访问（前端与后端不同源时必需）。
// allowOrigin 来自配置 ALLOW_ORIGIN：
//   - "*"     → 放行任意来源（默认，适合本地开发 / 多预览域名）
//   - 指定域名  → 仅放行该来源（生产推荐），不匹配则不设 Allow-Origin，浏览器会拦截
func cors(allowOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if allowOrigin == "" || allowOrigin == "*" {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin := c.GetHeader("Origin"); origin == allowOrigin {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// New 注册路由并返回 *gin.Engine。
// 注意：后端只提供 API；前端在同级 frontend/ 单独托管（Vercel / python http.server）。
// API 统一用 POST：避免 GET 被缓存/日志记录 ID、被 CSRF/img 预取误触发状态变更。
// GET /confirm/:id 是给人类点的邮件落地页（非 API），点按钮才 POST 心跳。
func New(svc *service.VaultService, cfg *config.Config) *gin.Engine {
	r := gin.Default()
	r.Use(cors(cfg.AllowOrigin))
	h := api.NewHandler(svc)

	r.POST("/api/vault", h.CreateOrUpdate)          // 创建 / 更新保险库
	r.POST("/api/vault/:id", h.Get)                 // 查询保险库（含密文）
	r.POST("/api/vault/:id/heartbeat", h.Heartbeat) // 主人报到
	r.POST("/api/vault/:id/trigger", h.Trigger)     // 查询触发状态并实时推进
	r.GET("/confirm/:id", h.ConfirmPage)            // 邮件确认存活落地页（人类页面）
	r.GET("/healthz", func(c *gin.Context) { // 保活/健康检查：让 Render 免费实例不被休眠，cron 才能持续跑
		c.JSON(200, gin.H{"status": "ok"})
	})

	return r
}
