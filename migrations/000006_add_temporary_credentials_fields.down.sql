-- 回滚临时凭证表字段添加

-- 删除索引
DROP INDEX IF EXISTS idx_temporary_credentials_token_type;
DROP INDEX IF EXISTS idx_temporary_credentials_role_arn;
DROP INDEX IF EXISTS idx_temporary_credentials_revoked_at;

-- 删除添加的字段
ALTER TABLE temporary_credentials 
DROP COLUMN IF EXISTS token_type,
DROP COLUMN IF EXISTS role_arn,
DROP COLUMN IF EXISTS role_session_name,
DROP COLUMN IF EXISTS external_id,
DROP COLUMN IF EXISTS session_policy,
DROP COLUMN IF EXISTS tags,
DROP COLUMN IF EXISTS duration_seconds,
DROP COLUMN IF EXISTS revoked_at;