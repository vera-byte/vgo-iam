-- 创建开发者认证表
CREATE TABLE developer_verifications (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    developer_type VARCHAR(20) NOT NULL CHECK (developer_type IN ('individual', 'enterprise')),
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'approved', 'rejected')) DEFAULT 'pending',
    
    -- 个人开发者信息
    real_name VARCHAR(100),
    id_card_number VARCHAR(20),
    id_card_front_url VARCHAR(500),
    id_card_back_url VARCHAR(500),
    
    -- 企业开发者信息
    company_name VARCHAR(200),
    business_license_number VARCHAR(50),
    business_license_url VARCHAR(500),
    legal_representative VARCHAR(100),
    company_address TEXT,
    
    -- 审核信息
    reviewer_id INTEGER REFERENCES users(id),
    review_comment TEXT,
    reviewed_at TIMESTAMP,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建应用表
CREATE TABLE applications (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    app_name VARCHAR(100) NOT NULL,
    app_description TEXT,
    app_type VARCHAR(20) NOT NULL CHECK (app_type IN ('web', 'mobile', 'desktop', 'api')),
    app_icon_url VARCHAR(500),
    app_website VARCHAR(200),
    status VARCHAR(20) NOT NULL CHECK (status IN ('active', 'inactive', 'suspended')) DEFAULT 'active',
    
    -- 应用配置
    callback_urls TEXT[], -- 回调URL列表
    allowed_origins TEXT[], -- 允许的域名列表
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 更新访问密钥表，关联应用
ALTER TABLE access_keys ADD COLUMN app_id INTEGER REFERENCES applications(id) ON DELETE CASCADE;
ALTER TABLE access_keys ADD COLUMN description VARCHAR(200);

-- 创建索引
CREATE INDEX idx_developer_verifications_user ON developer_verifications(user_id);
CREATE INDEX idx_developer_verifications_status ON developer_verifications(status);
CREATE INDEX idx_applications_user ON applications(user_id);
CREATE INDEX idx_applications_status ON applications(status);
CREATE INDEX idx_access_keys_app ON access_keys(app_id);

-- 创建唯一约束
CREATE UNIQUE INDEX idx_developer_verifications_user_type ON developer_verifications(user_id, developer_type);
CREATE UNIQUE INDEX idx_applications_user_name ON applications(user_id, app_name);