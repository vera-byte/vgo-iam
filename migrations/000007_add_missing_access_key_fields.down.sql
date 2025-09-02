-- 回滚访问密钥表字段添加
DO $$
BEGIN
  -- 删除索引
  DROP INDEX IF EXISTS idx_access_keys_app_id;
  DROP INDEX IF EXISTS idx_access_keys_expires_at;
  
  -- 删除添加的字段
  IF EXISTS (SELECT 1 FROM information_schema.columns 
            WHERE table_name='access_keys' AND column_name='app_id') THEN
    ALTER TABLE access_keys DROP COLUMN app_id;
    RAISE NOTICE 'Dropped column app_id';
  END IF;
  
  IF EXISTS (SELECT 1 FROM information_schema.columns 
            WHERE table_name='access_keys' AND column_name='description') THEN
    ALTER TABLE access_keys DROP COLUMN description;
    RAISE NOTICE 'Dropped column description';
  END IF;
  
  IF EXISTS (SELECT 1 FROM information_schema.columns 
            WHERE table_name='access_keys' AND column_name='expires_at') THEN
    ALTER TABLE access_keys DROP COLUMN expires_at;
    RAISE NOTICE 'Dropped column expires_at';
  END IF;
  
  IF EXISTS (SELECT 1 FROM information_schema.columns 
            WHERE table_name='access_keys' AND column_name='last_rotated_at') THEN
    ALTER TABLE access_keys DROP COLUMN last_rotated_at;
    RAISE NOTICE 'Dropped column last_rotated_at';
  END IF;
  
END $$;