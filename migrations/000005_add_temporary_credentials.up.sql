-- 创建临时凭证表
CREATE TABLE temporary_credentials (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    access_key_id VARCHAR(20) NOT NULL UNIQUE,
    encrypted_secret_access_key VARCHAR(255) NOT NULL,
    encrypted_session_token VARCHAR(255) NOT NULL UNIQUE,
    policy_document JSONB,
    expires_at TIMESTAMP NOT NULL,
    status VARCHAR(10) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'expired', 'revoked')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_temporary_credentials_user ON temporary_credentials(user_id);
CREATE INDEX idx_temporary_credentials_access_key ON temporary_credentials(access_key_id);
CREATE INDEX idx_temporary_credentials_session_token ON temporary_credentials(encrypted_session_token);
CREATE INDEX idx_temporary_credentials_status ON temporary_credentials(status);
CREATE INDEX idx_temporary_credentials_expires_at ON temporary_credentials(expires_at);
CREATE INDEX idx_temporary_credentials_status_expires ON temporary_credentials(status, expires_at);

-- 添加注释
COMMENT ON TABLE temporary_credentials IS 'STS临时凭证表';
COMMENT ON COLUMN temporary_credentials.id IS '主键ID';
COMMENT ON COLUMN temporary_credentials.user_id IS '用户ID，关联users表';
COMMENT ON COLUMN temporary_credentials.access_key_id IS '临时访问密钥ID';
COMMENT ON COLUMN temporary_credentials.encrypted_secret_access_key IS '加密的临时访问密钥';
COMMENT ON COLUMN temporary_credentials.encrypted_session_token IS '加密的会话令牌';
COMMENT ON COLUMN temporary_credentials.policy_document IS '临时权限策略文档';
COMMENT ON COLUMN temporary_credentials.expires_at IS '过期时间';
COMMENT ON COLUMN temporary_credentials.status IS '状态：active-活跃，expired-已过期，revoked-已撤销';
COMMENT ON COLUMN temporary_credentials.created_at IS '创建时间';