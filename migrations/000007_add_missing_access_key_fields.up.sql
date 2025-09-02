-- 添加访问密钥表缺失的字段
DO $$
BEGIN
  -- 添加 app_id 字段（如果不存在）
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                WHERE table_name='access_keys' AND column_name='app_id') THEN
    ALTER TABLE access_keys ADD COLUMN app_id INTEGER;
    RAISE NOTICE 'Added column app_id';
  END IF;
  
  -- 添加 description 字段（如果不存在）
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                WHERE table_name='access_keys' AND column_name='description') THEN
    ALTER TABLE access_keys ADD COLUMN description TEXT DEFAULT '';
    RAISE NOTICE 'Added column description';
  END IF;
  
  -- 添加 expires_at 字段（如果不存在）
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                WHERE table_name='access_keys' AND column_name='expires_at') THEN
    ALTER TABLE access_keys ADD COLUMN expires_at TIMESTAMP WITH TIME ZONE;
    RAISE NOTICE 'Added column expires_at';
  END IF;
  
  -- 添加 last_rotated_at 字段（如果不存在）
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                WHERE table_name='access_keys' AND column_name='last_rotated_at') THEN
    ALTER TABLE access_keys ADD COLUMN last_rotated_at TIMESTAMP WITH TIME ZONE;
    RAISE NOTICE 'Added column last_rotated_at';
  END IF;
  
  -- 为现有记录设置默认的过期时间（3个月后）
  UPDATE access_keys 
  SET expires_at = created_at + INTERVAL '3 months' 
  WHERE expires_at IS NULL;
  
END $$;

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_access_keys_app_id ON access_keys(app_id);
CREATE INDEX IF NOT EXISTS idx_access_keys_expires_at ON access_keys(expires_at);