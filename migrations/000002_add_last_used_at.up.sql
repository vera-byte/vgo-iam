-- 为访问密钥添加最后使用时间字段
ALTER TABLE access_keys ADD COLUMN last_used_at TIMESTAMP;

-- 创建最后使用时间索引
CREATE INDEX idx_access_keys_last_used ON access_keys(last_used_at);