-- 创建用户表
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255), -- 预留字段，用于控制台登录
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建策略表
CREATE TABLE policies (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    policy_document JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建访问密钥表
CREATE TABLE access_keys (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    access_key_id VARCHAR(20) NOT NULL UNIQUE,
    encrypted_secret_access_key TEXT NOT NULL,
    status VARCHAR(10) NOT NULL CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 用户策略关联表
CREATE TABLE user_policies (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    policy_id INTEGER NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, policy_id)
);

-- 创建索引
CREATE INDEX idx_access_keys_user ON access_keys(user_id);
CREATE INDEX idx_access_keys_status ON access_keys(status);
CREATE INDEX idx_user_policies_user ON user_policies(user_id);
CREATE INDEX idx_user_policies_policy ON user_policies(policy_id);