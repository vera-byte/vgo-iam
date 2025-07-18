-- 确保列存在且类型正确
DO $$
BEGIN
  -- 添加列（如果不存在）
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                WHERE table_name='access_keys' AND column_name='last_used_at') THEN
    ALTER TABLE access_keys ADD COLUMN last_used_at TIMESTAMP WITH TIME ZONE;
    RAISE NOTICE 'Added column last_used_at';
  END IF;
  
  -- 更新现有记录的默认值
  UPDATE access_keys SET last_used_at = created_at WHERE last_used_at IS NULL;
END $$;