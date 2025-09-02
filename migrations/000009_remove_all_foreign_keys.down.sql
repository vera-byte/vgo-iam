-- 恢复外键约束
-- 注意：这个回滚操作可能会失败，如果数据不满足外键约束条件

DO $$
BEGIN
    -- 恢复 access_keys 表的 user_id 外键约束
    IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                  WHERE constraint_name = 'access_keys_user_id_fkey' 
                  AND table_name = 'access_keys') THEN
        ALTER TABLE access_keys 
        ADD CONSTRAINT access_keys_user_id_fkey 
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
        RAISE NOTICE 'Restored access_keys.user_id foreign key constraint';
    END IF;
    
    -- 恢复 user_policies 表的外键约束
    IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                  WHERE constraint_name = 'user_policies_user_id_fkey' 
                  AND table_name = 'user_policies') THEN
        ALTER TABLE user_policies 
        ADD CONSTRAINT user_policies_user_id_fkey 
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
        RAISE NOTICE 'Restored user_policies.user_id foreign key constraint';
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                  WHERE constraint_name = 'user_policies_policy_id_fkey' 
                  AND table_name = 'user_policies') THEN
        ALTER TABLE user_policies 
        ADD CONSTRAINT user_policies_policy_id_fkey 
        FOREIGN KEY (policy_id) REFERENCES policies(id) ON DELETE CASCADE;
        RAISE NOTICE 'Restored user_policies.policy_id foreign key constraint';
    END IF;
    
    -- 恢复 temporary_credentials 表的外键约束
    IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                  WHERE constraint_name = 'temporary_credentials_user_id_fkey' 
                  AND table_name = 'temporary_credentials') THEN
        ALTER TABLE temporary_credentials 
        ADD CONSTRAINT temporary_credentials_user_id_fkey 
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
        RAISE NOTICE 'Restored temporary_credentials.user_id foreign key constraint';
    END IF;
    
    RAISE NOTICE 'Foreign key constraints restoration completed';
    
EXCEPTION
    WHEN others THEN
        RAISE WARNING 'Failed to restore some foreign key constraints: %', SQLERRM;
        RAISE NOTICE 'This may be due to data integrity issues. Please check your data manually.';
END $$;

-- 移除注释
COMMENT ON TABLE users IS NULL;
COMMENT ON TABLE policies IS NULL;
COMMENT ON TABLE access_keys IS NULL;
COMMENT ON TABLE user_policies IS NULL;
COMMENT ON TABLE temporary_credentials IS 'STS临时凭证表';