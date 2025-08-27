-- 删除索引
DROP INDEX IF EXISTS idx_access_keys_app;
DROP INDEX IF EXISTS idx_applications_status;
DROP INDEX IF EXISTS idx_applications_user;
DROP INDEX IF EXISTS idx_developer_verifications_status;
DROP INDEX IF EXISTS idx_developer_verifications_user;
DROP INDEX IF EXISTS idx_applications_user_name;
DROP INDEX IF EXISTS idx_developer_verifications_user_type;

-- 删除访问密钥表的新增列
ALTER TABLE access_keys DROP COLUMN IF EXISTS description;
ALTER TABLE access_keys DROP COLUMN IF EXISTS app_id;

-- 删除应用表
DROP TABLE IF EXISTS applications;

-- 删除开发者认证表
DROP TABLE IF EXISTS developer_verifications;