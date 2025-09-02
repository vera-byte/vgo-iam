-- 为临时凭证表添加缺失的字段

-- 添加token_type字段
ALTER TABLE temporary_credentials 
ADD COLUMN token_type VARCHAR(20) NOT NULL DEFAULT 'session' 
CHECK (token_type IN ('session', 'assume_role'));

-- 添加角色相关字段
ALTER TABLE temporary_credentials 
ADD COLUMN role_arn VARCHAR(255),
ADD COLUMN role_session_name VARCHAR(64),
ADD COLUMN external_id VARCHAR(1224);

-- 添加策略和标签字段
ALTER TABLE temporary_credentials 
ADD COLUMN session_policy TEXT,
ADD COLUMN tags TEXT;

-- 添加持续时间字段
ALTER TABLE temporary_credentials 
ADD COLUMN duration_seconds INTEGER NOT NULL DEFAULT 3600;

-- 添加撤销时间字段
ALTER TABLE temporary_credentials 
ADD COLUMN revoked_at TIMESTAMP;

-- 创建新的索引
CREATE INDEX idx_temporary_credentials_token_type ON temporary_credentials(token_type);
CREATE INDEX idx_temporary_credentials_role_arn ON temporary_credentials(role_arn);
CREATE INDEX idx_temporary_credentials_revoked_at ON temporary_credentials(revoked_at);

-- 添加字段注释
COMMENT ON COLUMN temporary_credentials.token_type IS '令牌类型：session-会话令牌，assume_role-角色假设令牌';
COMMENT ON COLUMN temporary_credentials.role_arn IS '角色ARN（AssumeRole时使用）';
COMMENT ON COLUMN temporary_credentials.role_session_name IS '角色会话名称';
COMMENT ON COLUMN temporary_credentials.external_id IS '外部ID';
COMMENT ON COLUMN temporary_credentials.session_policy IS '会话策略';
COMMENT ON COLUMN temporary_credentials.tags IS '标签（JSON格式）';
COMMENT ON COLUMN temporary_credentials.duration_seconds IS '有效期（秒）';
COMMENT ON COLUMN temporary_credentials.revoked_at IS '撤销时间';