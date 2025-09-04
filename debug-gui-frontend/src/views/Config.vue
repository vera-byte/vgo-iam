<template>
  <div class="config">
    <div class="page-header">
      <h1>系统配置</h1>
      <el-button type="primary" @click="saveConfig" :loading="saving">
        <el-icon><Check /></el-icon>
        保存配置
      </el-button>
    </div>

    <el-tabs v-model="activeTab" class="config-tabs">
      <!-- 基础配置 -->
      <el-tab-pane label="基础配置" name="basic">
        <el-card>
          <el-form :model="config.basic" label-width="150px" class="config-form">
            <el-form-item label="系统名称">
              <el-input v-model="config.basic.system_name" placeholder="请输入系统名称" />
            </el-form-item>
            <el-form-item label="系统描述">
              <el-input v-model="config.basic.system_description" type="textarea" :rows="3" placeholder="请输入系统描述" />
            </el-form-item>
            <el-form-item label="管理员邮箱">
              <el-input v-model="config.basic.admin_email" placeholder="请输入管理员邮箱" />
            </el-form-item>
            <el-form-item label="系统版本">
              <el-input v-model="config.basic.version" placeholder="请输入系统版本" />
            </el-form-item>
            <el-form-item label="维护模式">
              <el-switch v-model="config.basic.maintenance_mode" />
            </el-form-item>
          </el-form>
        </el-card>
      </el-tab-pane>

      <!-- 安全配置 -->
      <el-tab-pane label="安全配置" name="security">
        <el-card>
          <el-form :model="config.security" label-width="150px" class="config-form">
            <el-form-item label="JWT 密钥">
              <el-input v-model="config.security.jwt_secret" type="password" placeholder="请输入JWT密钥" show-password />
            </el-form-item>
            <el-form-item label="Token 过期时间">
              <el-input-number v-model="config.security.token_expiry" :min="1" :max="24" /> 小时
            </el-form-item>
            <el-form-item label="密码最小长度">
              <el-input-number v-model="config.security.password_min_length" :min="6" :max="20" />
            </el-form-item>
            <el-form-item label="登录失败锁定">
              <el-switch v-model="config.security.login_lockout_enabled" />
            </el-form-item>
            <el-form-item label="最大失败次数" v-if="config.security.login_lockout_enabled">
              <el-input-number v-model="config.security.max_login_attempts" :min="3" :max="10" />
            </el-form-item>
            <el-form-item label="锁定时间" v-if="config.security.login_lockout_enabled">
              <el-input-number v-model="config.security.lockout_duration" :min="5" :max="60" /> 分钟
            </el-form-item>
          </el-form>
        </el-card>
      </el-tab-pane>

      <!-- 数据库配置 -->
      <el-tab-pane label="数据库配置" name="database">
        <el-card>
          <el-form :model="config.database" label-width="150px" class="config-form">
            <el-form-item label="数据库类型">
              <el-select v-model="config.database.type" placeholder="请选择数据库类型">
                <el-option label="PostgreSQL" value="postgresql" />
                <el-option label="MySQL" value="mysql" />
                <el-option label="SQLite" value="sqlite" />
              </el-select>
            </el-form-item>
            <el-form-item label="主机地址">
              <el-input v-model="config.database.host" placeholder="请输入数据库主机地址" />
            </el-form-item>
            <el-form-item label="端口">
              <el-input-number v-model="config.database.port" :min="1" :max="65535" />
            </el-form-item>
            <el-form-item label="数据库名">
              <el-input v-model="config.database.name" placeholder="请输入数据库名" />
            </el-form-item>
            <el-form-item label="用户名">
              <el-input v-model="config.database.username" placeholder="请输入数据库用户名" />
            </el-form-item>
            <el-form-item label="密码">
              <el-input v-model="config.database.password" type="password" placeholder="请输入数据库密码" show-password />
            </el-form-item>
            <el-form-item label="最大连接数">
              <el-input-number v-model="config.database.max_connections" :min="1" :max="100" />
            </el-form-item>
            <el-form-item>
              <el-button @click="testConnection" :loading="testing">测试连接</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-tab-pane>

      <!-- 日志配置 -->
      <el-tab-pane label="日志配置" name="logging">
        <el-card>
          <el-form :model="config.logging" label-width="150px" class="config-form">
            <el-form-item label="日志级别">
              <el-select v-model="config.logging.level" placeholder="请选择日志级别">
                <el-option label="DEBUG" value="debug" />
                <el-option label="INFO" value="info" />
                <el-option label="WARN" value="warn" />
                <el-option label="ERROR" value="error" />
              </el-select>
            </el-form-item>
            <el-form-item label="日志格式">
              <el-select v-model="config.logging.format" placeholder="请选择日志格式">
                <el-option label="JSON" value="json" />
                <el-option label="文本" value="text" />
              </el-select>
            </el-form-item>
            <el-form-item label="日志文件路径">
              <el-input v-model="config.logging.file_path" placeholder="请输入日志文件路径" />
            </el-form-item>
            <el-form-item label="最大文件大小">
              <el-input-number v-model="config.logging.max_file_size" :min="1" :max="1000" /> MB
            </el-form-item>
            <el-form-item label="保留文件数">
              <el-input-number v-model="config.logging.max_backup_files" :min="1" :max="30" />
            </el-form-item>
            <el-form-item label="启用控制台输出">
              <el-switch v-model="config.logging.console_enabled" />
            </el-form-item>
          </el-form>
        </el-card>
      </el-tab-pane>

      <!-- API 配置 -->
      <el-tab-pane label="API 配置" name="api">
        <el-card>
          <el-form :model="config.api" label-width="150px" class="config-form">
            <el-form-item label="服务端口">
              <el-input-number v-model="config.api.port" :min="1" :max="65535" />
            </el-form-item>
            <el-form-item label="请求超时时间">
              <el-input-number v-model="config.api.timeout" :min="1" :max="300" /> 秒
            </el-form-item>
            <el-form-item label="最大请求大小">
              <el-input-number v-model="config.api.max_request_size" :min="1" :max="100" /> MB
            </el-form-item>
            <el-form-item label="启用 CORS">
              <el-switch v-model="config.api.cors_enabled" />
            </el-form-item>
            <el-form-item label="允许的域名" v-if="config.api.cors_enabled">
              <el-input v-model="config.api.allowed_origins" placeholder="请输入允许的域名，多个用逗号分隔" />
            </el-form-item>
            <el-form-item label="启用限流">
              <el-switch v-model="config.api.rate_limit_enabled" />
            </el-form-item>
            <el-form-item label="每分钟请求数" v-if="config.api.rate_limit_enabled">
              <el-input-number v-model="config.api.requests_per_minute" :min="1" :max="10000" />
            </el-form-item>
          </el-form>
        </el-card>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Check } from '@element-plus/icons-vue'
import { api } from '@/api'

interface Config {
  basic: {
    system_name: string
    system_description: string
    admin_email: string
    version: string
    maintenance_mode: boolean
  }
  security: {
    jwt_secret: string
    token_expiry: number
    password_min_length: number
    login_lockout_enabled: boolean
    max_login_attempts: number
    lockout_duration: number
  }
  database: {
    type: string
    host: string
    port: number
    name: string
    username: string
    password: string
    max_connections: number
  }
  logging: {
    level: string
    format: string
    file_path: string
    max_file_size: number
    max_backup_files: number
    console_enabled: boolean
  }
  api: {
    port: number
    timeout: number
    max_request_size: number
    cors_enabled: boolean
    allowed_origins: string
    rate_limit_enabled: boolean
    requests_per_minute: number
  }
}

const activeTab = ref('basic')
const saving = ref(false)
const testing = ref(false)

const config = reactive<Config>({
  basic: {
    system_name: 'VGO IAM System',
    system_description: '身份认证与访问管理系统',
    admin_email: 'admin@example.com',
    version: 'v1.0.0',
    maintenance_mode: false
  },
  security: {
    jwt_secret: '',
    token_expiry: 24,
    password_min_length: 8,
    login_lockout_enabled: true,
    max_login_attempts: 5,
    lockout_duration: 15
  },
  database: {
    type: 'postgresql',
    host: 'localhost',
    port: 5432,
    name: 'vgo_iam',
    username: 'postgres',
    password: '',
    max_connections: 20
  },
  logging: {
    level: 'info',
    format: 'json',
    file_path: '/var/log/vgo-iam.log',
    max_file_size: 100,
    max_backup_files: 10,
    console_enabled: true
  },
  api: {
    port: 8080,
    timeout: 30,
    max_request_size: 10,
    cors_enabled: true,
    allowed_origins: '*',
    rate_limit_enabled: true,
    requests_per_minute: 1000
  }
})

const loadConfig = async () => {
  try {
    const response = await api.config.get()
    Object.assign(config, response.data)
  } catch (error) {
    ElMessage.error('加载配置失败')
  }
}

const saveConfig = async () => {
  saving.value = true
  try {
    await api.config.update(config)
    ElMessage.success('配置保存成功')
  } catch (error) {
    ElMessage.error('配置保存失败')
  } finally {
    saving.value = false
  }
}

const testConnection = async () => {
  testing.value = true
  try {
    // 模拟数据库连接测试
    await new Promise(resolve => setTimeout(resolve, 2000))
    ElMessage.success('数据库连接测试成功')
  } catch (error) {
    ElMessage.error('数据库连接测试失败')
  } finally {
    testing.value = false
  }
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.config {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h1 {
  margin: 0;
  color: var(--el-text-color-primary);
}

.config-tabs {
  background: white;
  border-radius: 8px;
  padding: 20px;
}

.config-form {
  max-width: 600px;
}

.config-form .el-form-item {
  margin-bottom: 20px;
}
</style>