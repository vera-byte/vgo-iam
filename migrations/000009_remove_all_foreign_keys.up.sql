-- 移除所有外键约束
-- 这个迁移将移除数据库中的所有外键约束，保持数据完整性由应用层控制

DO $$
DECLARE
    constraint_record RECORD;
BEGIN
    -- 查找并删除所有外键约束
    FOR constraint_record IN 
        SELECT 
            tc.table_name,
            tc.constraint_name
        FROM information_schema.table_constraints tc
        WHERE tc.constraint_type = 'FOREIGN KEY'
        AND tc.table_schema = 'public'
    LOOP
        EXECUTE format('ALTER TABLE %I DROP CONSTRAINT IF EXISTS %I', 
                      constraint_record.table_name, 
                      constraint_record.constraint_name);
        RAISE NOTICE 'Dropped foreign key constraint: %.%', 
                     constraint_record.table_name, 
                     constraint_record.constraint_name;
    END LOOP;
    
    RAISE NOTICE 'All foreign key constraints have been removed';
END $$;

-- 添加注释说明数据完整性现在由应用层维护
COMMENT ON TABLE users IS '用户表 - 数据完整性由应用层维护';
COMMENT ON TABLE policies IS '策略表 - 数据完整性由应用层维护';
COMMENT ON TABLE access_keys IS '访问密钥表 - 数据完整性由应用层维护';
COMMENT ON TABLE user_policies IS '用户策略关联表 - 数据完整性由应用层维护';
COMMENT ON TABLE temporary_credentials IS 'STS临时凭证表 - 数据完整性由应用层维护';