-- 安全删除列（仅当列存在时）
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns 
             WHERE table_name='access_keys' AND column_name='last_used_at') THEN
    ALTER TABLE access_keys DROP COLUMN last_used_at;
    RAISE NOTICE 'Dropped column last_used_at';
  ELSE
    RAISE NOTICE 'Column last_used_at does not exist, skipping';
  END IF;
END $$;