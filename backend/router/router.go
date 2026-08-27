package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stridedot/family-password-vault/backend/api"
	"github.com/stridedot/family-password-vault/backend/service"
)

// cors 允许前端跨域访问（前端与后端不同源时必需）。
// 注意：此处 Allow-Origin 为 *，仅适合演示；生产应限制为前端域名。
func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
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
func New(svc *service.VaultService) *gin.Engine {
	r := gin.Default()
	r.Use(cors())
	h := api.NewHandler(svc)

	r.POST("/api/vault", h.CreateOrUpdate)     // 创建 / 更新保险库
	r.POST("/api/vault/:id", h.Get)            // 查询保险库（含密文）
	r.POST("/api/vault/:id/heartbeat", h.Heartbeat) // 主人报到
	r.POST("/api/vault/:id/trigger", h.Trigger) // 查询触发状态并实时推进
	r.GET("/confirm/:id", h.ConfirmPage)       // 邮件确认存活落地页（人类页面）

	return r
}
