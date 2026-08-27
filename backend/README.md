# 家庭密码 / 文档继承保险库（family-password-vault）

> 国内版家庭数字遗产保险库：把密码、账号、关键文档存进**只有自己能解开、死后家人能按简单步骤拿到**的保险库。正确解法是零知识（客户端加密 + 死亡开关 + 密钥分片），服务器永远看不到明文——不是"共享账号"。

本仓库是「家庭密码」项目的**代码根**（前端 + 后端）。产品需求 / PRD / 零知识架构说明在仓库外的实验目录（`../需求文档.md`、`../PRD.md`）。

## 架构（零知识）

- 加密全在**客户端**：Web Crypto `AES-GCM` 加密 + `PBKDF2` 从「用户密码 + 随机盐」派生主密钥 K。
- 服务器只存**密文 + 拆散的密钥分片**，永远见不到明文、见不到 K——这是产品成立的根本，也是合规护身符。
- 死亡开关：用户定时「心跳报到」；连续 N 天未报到 → 进入释放模式，留「反悔窗口」供用户一键取消；超时后受益人凭释放密码取走密文并在本地解密。

```
family-password-vault/
├── frontend/                # 纯前端 H5（Web Crypto 本地加密，零后端即可演示）
│   ├── index.html
│   ├── style.css
│   └── app.js
└── backend/                 # Golang 后端（gin + gorm + cron）
    ├── main.go
    ├── config/              # 配置（端口 / DB 路径 / cron 表达式）
    ├── model/               # 实体
    ├── repository/          # gorm 存储
    ├── service/             # 业务逻辑
    ├── api/                 # gin handlers
    ├── router/              # 路由
    └── scheduler/           # 定时
```

## 本地运行

### 前端（零后端 Demo）
> Web Crypto 需安全上下文，`file://` 直接打开会拿不到 `crypto.subtle`，必须走 localhost / https。

```bash
cd frontend
python -m http.server 8000
# 浏览器打开 http://localhost:8000
```

演示闭环：填主密码 + 释放密码创建保险库 → 存条目并「我还在」报到 → 把反悔窗口设小后「模拟失联」→ 受益人视图输释放密码取走（窗口内主人可取消）。

### 后端（Golang）

API 概览：
- `PUT  /api/vault`             创建 / 更新保险库（密文 + 受益人密文 + 静默/宽限期毫秒，演示可覆盖）
- `GET  /api/vault/:id`         取回保险库密文（仅失联进入释放后返回受益人可取内容）
- `POST /api/vault/:id/heartbeat`  主人心跳报到（取消释放）
- `GET  /api/vault/:id/trigger`    查询释放状态（none / grace / released）

> 受益人取用在**客户端本地**完成：拿到密文 + 密钥分片后，用释放密码在浏览器里解密。服务端**没有 `/release` 接口**——零知识原则，服务器永远不接触明文。

## 部署

- **前端 → Vercel**：项目 `Root Directory = frontend`（纯静态，复用 `vercel.json`）。
- **后端 → Railway / Render**：项目 `Root Directory = backend`（平台在 `backend/` 找 `go.mod` 构建常驻进程，跑心跳 scheduler）。
- 全国外、不备案最快落地；零知识存密文，页面加「数据加密存储于境外」告知。

