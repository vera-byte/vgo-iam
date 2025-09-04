<template>
  <div class="signature-config-container">
    <div class="page-header">
      <h1>签名配置</h1>
      <el-button type="primary" @click="saveConfig" :loading="saving">
        <el-icon><Check /></el-icon>
        保存配置
      </el-button>
    </div>

    <div class="content-card">
      <el-alert
        title="重要提示"
        type="warning"
        description="请妥善保管您的访问密钥信息，不要在不安全的环境中输入或存储。"
        show-icon
        :closable="false"
        style="margin-bottom: 20px"
      />

      <el-form :model="configForm" :rules="configRules" ref="configFormRef" label-width="150px">
        <el-form-item label="Access Key ID" prop="accessKeyId">
          <el-input
            v-model="configForm.accessKeyId"
            placeholder="请输入 Access Key ID"
            clearable
          />
        </el-form-item>
        
        <el-form-item label="Secret Access Key" prop="secretAccessKey">
          <el-input
            v-model="configForm.secretAccessKey"
            type="password"
            placeholder="请输入 Secret Access Key"
            show-password
            clearable
          />
        </el-form-item>
      </el-form>

      <div class="current-config" v-if="currentConfig.accessKeyId">
        <h3>当前配置</h3>
        <div class="config-info">
          <div class="info-item">
            <label>Access Key ID:</label>
            <span>{{ currentConfig.accessKeyId }}</span>
          </div>
          <div class="info-item">
            <label>Secret Access Key:</label>
            <span>{{ maskSecretKey(currentConfig.secretAccessKey) }}</span>
          </div>
        </div>
        <el-button type="danger" @click="clearConfig" style="margin-top: 16px">
          <el-icon><Delete /></el-icon>
          清除配置
        </el-button>
      </div>

      <div class="no-config" v-else>
        <el-empty description="暂未配置签名信息">
          <el-button type="primary" @click="showQuickSetup = true">
            快速设置
          </el-button>
        </el-empty>
      </div>
    </div>

    <!-- 快速设置对话框 -->
    <el-dialog v-model="showQuickSetup" title="快速设置" width="500px">
      <el-alert
        title="管理员密钥"
        type="info"
        description="如果您是系统管理员，可以使用以下预设的管理员密钥进行快速配置。"
        show-icon
        :closable="false"
        style="margin-bottom: 20px"
      />
      
      <div class="admin-keys">
        <div class="key-item">
          <label>Access Key ID:</label>
          <el-input v-model="adminKeys.accessKeyId" readonly>
            <template #append>
              <el-button @click="copyToClipboard(adminKeys.accessKeyId)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </template>
          </el-input>
        </div>
        <div class="key-item">
          <label>Secret Access Key:</label>
          <el-input v-model="adminKeys.secretAccessKey" readonly>
            <template #append>
              <el-button @click="copyToClipboard(adminKeys.secretAccessKey)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </template>
          </el-input>
        </div>
      </div>
      
      <template #footer>
        <el-button @click="showQuickSetup = false">取消</el-button>
        <el-button type="primary" @click="useAdminKeys">
          使用管理员密钥
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Check, Delete, CopyDocument } from '@element-plus/icons-vue'
import { getSignatureConfig, setSignatureConfig, clearSignatureConfig } from '../utils/signature'

interface ConfigForm {
  accessKeyId: string
  secretAccessKey: string
}

const saving = ref(false)
const showQuickSetup = ref(false)
const configFormRef = ref()

const configForm = ref<ConfigForm>({
  accessKeyId: '',
  secretAccessKey: ''
})

const currentConfig = ref<ConfigForm>({
  accessKeyId: '',
  secretAccessKey: ''
})

// 管理员密钥（从之前获取的密钥）
const adminKeys = ref({
  accessKeyId: 'Ga0rTSg3NSyoOkFUx9jg',
  secretAccessKey: 'poh7b4bQi9fwXfIPXVGMzF0qiqaf9gDI9drEXtpk'
})

const configRules = {
  accessKeyId: [
    { required: true, message: '请输入 Access Key ID', trigger: 'blur' },
    { min: 10, message: 'Access Key ID 长度不能少于 10 位', trigger: 'blur' }
  ],
  secretAccessKey: [
    { required: true, message: '请输入 Secret Access Key', trigger: 'blur' },
    { min: 20, message: 'Secret Access Key 长度不能少于 20 位', trigger: 'blur' }
  ]
}

/**
 * 加载当前配置
 */
const loadCurrentConfig = () => {
  const config = getSignatureConfig()
  if (config) {
    currentConfig.value = {
      accessKeyId: config.accessKeyId,
      secretAccessKey: config.secretAccessKey
    }
  }
}

/**
 * 保存配置
 */
const saveConfig = async () => {
  if (!configFormRef.value) return
  
  try {
    await configFormRef.value.validate()
    saving.value = true
    
    setSignatureConfig({
      accessKeyId: configForm.value.accessKeyId,
      secretAccessKey: configForm.value.secretAccessKey
    })
    
    loadCurrentConfig()
    
    // 清空表单
    configForm.value = {
      accessKeyId: '',
      secretAccessKey: ''
    }
    
    ElMessage.success('签名配置保存成功')
  } catch (error) {
    console.error('保存配置失败:', error)
    ElMessage.error('保存配置失败')
  } finally {
    saving.value = false
  }
}

/**
 * 清除配置
 */
const clearConfig = async () => {
  try {
    await ElMessageBox.confirm(
      '确定要清除当前的签名配置吗？清除后将无法访问 API。',
      '确认清除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    clearSignatureConfig()
    loadCurrentConfig()
    ElMessage.success('签名配置已清除')
  } catch (error) {
    if (error !== 'cancel') {
      console.error('清除配置失败:', error)
      ElMessage.error('清除配置失败')
    }
  }
}

/**
 * 使用管理员密钥
 */
const useAdminKeys = () => {
  configForm.value = {
    accessKeyId: adminKeys.value.accessKeyId,
    secretAccessKey: adminKeys.value.secretAccessKey
  }
  showQuickSetup.value = false
  ElMessage.success('已填入管理员密钥，请点击保存配置')
}

/**
 * 复制到剪贴板
 */
const copyToClipboard = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制到剪贴板')
  } catch (error) {
    console.error('复制失败:', error)
    ElMessage.error('复制失败')
  }
}

/**
 * 掩码显示密钥
 */
const maskSecretKey = (key: string) => {
  if (!key) return ''
  if (key.length <= 8) return '*'.repeat(key.length)
  return key.substring(0, 4) + '*'.repeat(key.length - 8) + key.substring(key.length - 4)
}

onMounted(() => {
  loadCurrentConfig()
})
</script>

<style scoped>
.signature-config-container {
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

.content-card {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.current-config {
  margin-top: 30px;
  padding: 20px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
}

.current-config h3 {
  margin: 0 0 16px 0;
  color: var(--el-text-color-primary);
}

.config-info {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.info-item label {
  width: 150px;
  font-weight: bold;
  color: var(--el-text-color-primary);
  flex-shrink: 0;
}

.info-item span {
  font-family: monospace;
  background: var(--el-bg-color);
  padding: 4px 8px;
  border-radius: 4px;
  border: 1px solid var(--el-border-color);
}

.no-config {
  margin-top: 30px;
}

.admin-keys {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.key-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.key-item label {
  font-weight: bold;
  color: var(--el-text-color-primary);
}
</style>