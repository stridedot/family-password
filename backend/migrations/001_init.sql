-- 家庭密码后端：vaults 表（零知识：只存密文 + 心跳 + 状态机）
-- 对应 model.Vault（TableName = "vaults"）
-- id 为客户端生成的 UUID v4（能力令牌），服务端不产 id；服务器全程不见明文/主密钥 K

CREATE TABLE IF NOT EXISTS vaults (
    id              UUID         PRIMARY KEY,               -- 客户端 crypto.randomUUID() 生成
    salt            TEXT         NOT NULL DEFAULT '',       -- base64，客户端 PBKDF2 派生主密钥用
    vault           TEXT         NOT NULL DEFAULT '',       -- base64 JSON{iv,ct}，主密码加密的密文
    beneficiary     TEXT         NOT NULL DEFAULT '',       -- base64 JSON{iv,ct}，释放密码加密的密文
    email           TEXT         NOT NULL DEFAULT '',       -- 主人通知邮箱（仅通知用途，非账户/登录）
    beneficiary_email TEXT        NOT NULL DEFAULT '',       -- 受益人邮箱：释放时自动通知TA来取用（仅元数据/PII，不存密文）
    heartbeat_at    BIGINT       NOT NULL DEFAULT 0,        -- 上次"我还活着"时间戳（unix ms）
    reminder_sent   BOOLEAN      NOT NULL DEFAULT false,    -- 本轮静默是否已发过预警邮件（防每小时重复发）
    grace_reminder_sent BOOLEAN   NOT NULL DEFAULT false,    -- 宽限期内是否已发过最终提醒（防重复发）
    trigger_status  VARCHAR(16)  NOT NULL DEFAULT 'none',   -- none | grace | released
    grace_ends_at   BIGINT       NOT NULL DEFAULT 0,        -- 宽限期结束时间戳（unix ms）
    silence_ms      BIGINT       NOT NULL DEFAULT 0,        -- 静默阈值（ms，默认 30 天）
    grace_ms        BIGINT       NOT NULL DEFAULT 0,        -- 宽限期长度（ms，默认 14 天=336h）
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),    -- 审计（gorm 自动填充）
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()     -- 审计（gorm 自动填充）
);

