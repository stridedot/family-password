package api

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/stridedot/family-password-vault/backend/model"
	"github.com/stridedot/family-password-vault/backend/service"
)

// Handler 持有 service，处理 HTTP 请求。只做 绑定/校验/响应，业务在 service。
type Handler struct {
	svc *service.VaultService
}

func NewHandler(svc *service.VaultService) *Handler {
	return &Handler{svc: svc}
}

// CreateOrUpdate PUT /api/vault —— 创建或更新保险库（密文上送服务端）
func (h *Handler) CreateOrUpdate(c *gin.Context) {
	var v model.Vault
	if err := c.ShouldBindJSON(&v); err != nil {
		badRequest(c, "bad json")
		return
	}
	if v.ID == "" {
		badRequest(c, "id required")
		return
	}
	// 默认值（silence_ms / grace_ms / trigger_status）由 service 层按配置补齐；
	// 已 released 的库拒绝写入（主人不应在受益人可取走后继续改密文）。
	if err := h.svc.Put(&v); err != nil {
		if errors.Is(err, service.ErrReleased) {
			unprocessable(c, "vault already released")
			return
		}
		internal(c, err.Error())
		return
	}
	ok(c, gin.H{"status": "ok"})
}

// Get GET /api/vault/:id —— 纯查询，返回记录（不推进状态机，状态机由 scheduler 推进）
// 注意：响应脱敏，不返回 email / reminder_sent / 阈值（尤其避免把主人邮箱泄露给受益人）。
func (h *Handler) Get(c *gin.Context) {
	v, err := h.svc.Get(c.Param("id"))
	if err != nil {
		notFound(c, "vault not found")
		return
	}
	ok(c, gin.H{
		"id":            v.ID,
		"salt":          v.Salt,
		"vault":         v.Vault,
		"beneficiary":   v.Beneficiary,
		"heartbeat_at":  v.HeartbeatAt,
		"trigger_status": v.TriggerStatus,
		"grace_ends_at": v.GraceEndsAt,
	})
}

// Heartbeat POST /api/vault/:id/heartbeat —— 主人报到（刷新心跳 + 取消释放）
func (h *Handler) Heartbeat(c *gin.Context) {
	v, err := h.svc.Heartbeat(c.Param("id"))
	if err != nil {
		notFound(c, "vault not found")
		return
	}
	ok(c, v)
}

// ConfirmPage GET /confirm/:id —— 邮件"确认存活"落地页（人类页面，非 API）
// 仅展示一个按钮，点击后才 POST /api/vault/:id/heartbeat 真正报到。
// 这样邮件扫描器/预览（只发 GET）不会误触发心跳，避免把"续命"暴露在邮件管道里。
func (h *Handler) ConfirmPage(c *gin.Context) {
	html := `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<title>家庭密码 · 确认存活</title>` +
		`<style>body{font-family:system-ui,-apple-system,sans-serif;max-width:460px;margin:48px auto;padding:0 16px;color:#1a1a1a;line-height:1.6}` +
		`h2{margin-bottom:8px}.muted{color:#666;font-size:14px}#btn{margin-top:20px;padding:12px 20px;font-size:15px;border:0;border-radius:10px;background:#2f6df6;color:#fff;cursor:pointer}` +
		`#btn:disabled{opacity:.6;cursor:default}.ok{color:#1a9d54}.err{color:#d23b3b}</style></head>` +
		`<body><h2>家庭密码 · 确认存活</h2>` +
		`<p id="msg" class="muted">点击下方按钮，确认你仍然在世，保险库将不会被释放。</p>` +
		`<button id="btn" onclick="confirmAlive()">确认我还活着</button>` +
		`<script>function confirmAlive(){var id=location.pathname.split('/').pop();var b=document.getElementById('btn');b.disabled=true;` +
		`fetch('/api/vault/'+id+'/heartbeat',{method:'POST'}).then(function(r){return r.ok;}).then(function(ok){` +
		`var m=document.getElementById('msg');if(ok){m.className='ok';m.textContent='✅ 已确认存活，保险库不会释放。可以关闭此页面了。';b.textContent='已确认';}` +
		`else{m.className='err';m.textContent='❌ 确认失败：保险库不存在或已释放。';b.disabled=false;}}).catch(function(e){` +
		`var m=document.getElementById('msg');m.className='err';m.textContent='❌ 网络错误：'+e.message;b.disabled=false;});}` +
		`</script></body></html>`
	c.Data(200, "text/html; charset=utf-8", []byte(html))
}

// Trigger GET /api/vault/:id/trigger —— 查询触发状态并实时推进（保证返回的是最新状态）
func (h *Handler) Trigger(c *gin.Context) {
	v, _, err := h.svc.Touch(c.Param("id"))
	if err != nil {
		notFound(c, "vault not found")
		return
	}
	ok(c, gin.H{
		"id":             v.ID,
		"trigger_status": v.TriggerStatus,
		"heartbeat_at":   v.HeartbeatAt,
		"grace_ends_at":  v.GraceEndsAt,
	})
}
