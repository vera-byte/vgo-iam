#!/bin/bash

# 添加管理员用户的脚本

# 设置数据库连接参数
DB_HOST="10.0.0.200"
DB_PORT="5432"
DB_NAME="vgo_iam"
DB_USER="vgo_iam"
DB_PASS="KESdCZeYYXBZcebH"

# 检查PostgreSQL是否运行
if ! pg_isready -h 10.0.0.200 -p 5432 > /dev/null 2>&1; then
    echo "PostgreSQL 没有在 10.0.0.200:5432 上运行"
    exit 1
fi

# 检查admin用户是否存在
ADMIN_EXISTS=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT 1 FROM users WHERE name = 'admin';" | xargs)

if [ "$ADMIN_EXISTS" = "1" ]; then
    echo "admin用户已存在"
else
    # 插入admin用户
    psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
    INSERT INTO users (name, display_name, email, created_at, updated_at)
    VALUES ('admin', 'System Administrator', 'admin@example.com', NOW(), NOW());
    "
    echo "已创建admin用户"
fi

# 获取admin用户的ID
ADMIN_ID=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT id FROM users WHERE name = 'admin';" | xargs)

# 检查admin策略是否存在
POLICY_EXISTS=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT 1 FROM policies WHERE name = 'admin-policy';" | xargs)

if [ "$POLICY_EXISTS" != "1" ]; then
    # 插入admin策略
    psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
    INSERT INTO policies (name, description, policy_document, created_at, updated_at)
    VALUES ('admin-policy', 'Administrator policy with full permissions', '{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Action\":\"*\",\"Resource\":\"*\"}]}', NOW(), NOW());
    "
    echo "已创建admin策略"
fi

# 获取admin策略的ID
POLICY_ID=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT id FROM policies WHERE name = 'admin-policy';" | xargs)

# 关联用户和策略
USER_POLICY_EXISTS=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT 1 FROM user_policies WHERE user_id = $ADMIN_ID AND policy_id = $POLICY_ID;" | xargs)

if [ "$USER_POLICY_EXISTS" != "1" ]; then
    psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
    INSERT INTO user_policies (user_id, policy_id) VALUES ($ADMIN_ID, $POLICY_ID);
    "
    echo "已关联admin用户和策略"
fi

# 生成访问密钥ID和密钥
ACCESS_KEY_ID="AKIA$(openssl rand -hex 10)"
SECRET_KEY=$(openssl rand -hex 32)

# 检查访问密钥是否已存在
KEY_EXISTS=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT 1 FROM access_keys WHERE access_key_id = '$ACCESS_KEY_ID';" | xargs)

if [ "$KEY_EXISTS" != "1" ]; then
    # 插入访问密钥（注意：实际生产环境应该加密密钥）
    psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
    INSERT INTO access_keys (user_id, access_key_id, encrypted_secret_access_key, status, created_at, updated_at)
    VALUES ($ADMIN_ID, '$ACCESS_KEY_ID', '$SECRET_KEY', 'active', NOW(), NOW());
    "
    echo "已创建admin访问密钥"
fi

echo "管理员用户初始化完成"
echo "访问密钥ID: $ACCESS_KEY_ID"
echo "密钥: $SECRET_KEY"
echo ""
echo "你可以使用这些凭据进行API调用"