<template>
  <div class="permissions-container">
    <div class="page-header">
      <h1>权限检查</h1>
    </div>

    <div class="content-card">
      <div class="check-form">
        <el-form :model="checkForm" :rules="checkRules" ref="checkFormRef" label-width="120px">
          <el-form-item label="用户ID" prop="user_id">
            <el-input v-model="checkForm.user_id" placeholder="请输入用户ID" />
          </el-form-item>
          <el-form-item label="资源" prop="resource">
            <el-input v-model="checkForm.resource" placeholder="请输入资源路径，如：/api/users" />
          </el-form-item>
          <el-form-item label="操作" prop="action">
            <el-select v-model="checkForm.action" placeholder="请选择操作">
              <el-option label="读取 (GET)" value="GET" />
              <el-option label="创建 (POST)" value="POST" />
              <el-option label="更新 (PUT)" value="PUT" />
              <el-option label="删除 (DELETE)" value="DELETE" />
              <el-option label="补丁 (PATCH)" value="PATCH" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="checkPermission" :loading="checking">
              检查权限
            </el-button>
            <el-button @click="resetForm">重置</el-button>
          </el-form-item>
        </el-form>
      </div>

      <div v-if="checkResult" class="result-section">
        <h3>检查结果</h3>
        <div class="result-card" :class="{ 'allowed': checkResult.allowed, 'denied': !checkResult.allowed }">
          <div class="result-header">
            <el-icon v-if="checkResult.allowed" class="success-icon"><Check /></el-icon>
            <el-icon v-else class="error-icon"><Close /></el-icon>
            <span class="result-text">
              {{ checkResult.allowed ? '权限允许' : '权限拒绝' }}
            </span>
          </div>
          
          <div class="result-details">
            <div class="detail-item">
              <label>用户ID:</label>
              <span>{{ checkResult.user_id }}</span>
            </div>
            <div class="detail-item">
              <label>资源:</label>
              <span>{{ checkResult.resource }}</span>
            </div>
            <div class="detail-item">
              <label>操作:</label>
              <span>{{ checkResult.action }}</span>
            </div>
            <div class="detail-item">
              <label>检查时间:</label>
              <span>{{ formatDate(checkResult.checked_at) }}</span>
            </div>
            <div v-if="checkResult.reason" class="detail-item">
              <label>原因:</label>
              <span>{{ checkResult.reason }}</span>
            </div>
            <div v-if="checkResult.matched_policies && checkResult.matched_policies.length > 0" class="detail-item">
              <label>匹配的策略:</label>
              <div class="policy-list">
                <el-tag 
                  v-for="policy in checkResult.matched_policies" 
                  :key="policy.id"
                  :type="policy.effect === 'allow' ? 'success' : 'danger'"
                >
                  {{ policy.name }} ({{ policy.effect }})
                </el-tag>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="history-section">
        <div class="section-header">
          <h3>检查历史</h3>
          <el-button size="small" @click="clearHistory">清空历史</el-button>
        </div>
        
        <el-table :data="checkHistory" stripe>
          <el-table-column prop="user_id" label="用户ID" width="150" />
          <el-table-column prop="resource" label="资源" width="200" />
          <el-table-column prop="action" label="操作" width="100" />
          <el-table-column prop="allowed" label="结果" width="100">
            <template #default="{ row }">
              <el-tag :type="row.allowed ? 'success' : 'danger'">
                {{ row.allowed ? '允许' : '拒绝' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="checked_at" label="检查时间" width="180">
            <template #default="{ row }">
              {{ formatDate(row.checked_at) }}
            </template>
          </el-table-column>
          <el-table-column prop="reason" label="原因" />
        </el-table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Check, Close } from '@element-plus/icons-vue'
import { api } from '../api'
import * as dayjs from 'dayjs'

interface PermissionCheckForm {
  user_id: string
  resource: string
  action: string
}

interface PermissionCheckResult {
  user_id: string
  resource: string
  action: string
  allowed: boolean
  reason?: string
  checked_at: string
  matched_policies?: {
    id: string
    name: string
    effect: 'allow' | 'deny'
  }[]
}

const checking = ref(false)
const checkFormRef = ref()
const checkResult = ref<PermissionCheckResult | null>(null)
const checkHistory = ref<PermissionCheckResult[]>([])

const checkForm = ref<PermissionCheckForm>({
  user_id: '',
  resource: '',
  action: ''
})

const checkRules = {
  user_id: [{ required: true, message: '请输入用户ID', trigger: 'blur' }],
  resource: [{ required: true, message: '请输入资源路径', trigger: 'blur' }],
  action: [{ required: true, message: '请选择操作', trigger: 'change' }]
}

const formatDate = (dateString: string) => {
  return dayjs(dateString).format('YYYY-MM-DD HH:mm:ss')
}

const checkPermission = async () => {
  if (!checkFormRef.value) return
  
  try {
    await checkFormRef.value.validate()
    
    checking.value = true
    
    const response = await api.permissions.check({
      user_id: checkForm.value.user_id,
      resource: checkForm.value.resource,
      action: checkForm.value.action
    })
    
    const result: PermissionCheckResult = {
      ...response.data,
      checked_at: new Date().toISOString()
    }
    
    checkResult.value = result
    
    // 添加到历史记录
    checkHistory.value.unshift(result)
    
    // 保持历史记录不超过50条
    if (checkHistory.value.length > 50) {
      checkHistory.value = checkHistory.value.slice(0, 50)
    }
    
    // 保存到本地存储
    localStorage.setItem('permission_check_history', JSON.stringify(checkHistory.value))
    
    ElMessage.success('权限检查完成')
  } catch (error) {
    console.error('权限检查失败:', error)
    ElMessage.error('权限检查失败')
  } finally {
    checking.value = false
  }
}

const resetForm = () => {
  checkForm.value = {
    user_id: '',
    resource: '',
    action: ''
  }
  checkResult.value = null
}

const clearHistory = () => {
  checkHistory.value = []
  localStorage.removeItem('permission_check_history')
  ElMessage.success('历史记录已清空')
}

const loadHistory = () => {
  const saved = localStorage.getItem('permission_check_history')
  if (saved) {
    try {
      checkHistory.value = JSON.parse(saved)
    } catch (error) {
      console.error('加载历史记录失败:', error)
    }
  }
}

onMounted(() => {
  loadHistory()
})
</script>

<style scoped>
.permissions-container {
  padding: 20px;
}

.page-header {
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

.check-form {
  margin-bottom: 30px;
  padding: 20px;
  background: var(--el-bg-color-page);
  border-radius: 8px;
}

.result-section {
  margin-bottom: 30px;
}

.result-section h3 {
  margin-bottom: 16px;
  color: var(--el-text-color-primary);
}

.result-card {
  padding: 20px;
  border-radius: 8px;
  border: 2px solid;
}

.result-card.allowed {
  border-color: var(--el-color-success);
  background: var(--el-color-success-light-9);
}

.result-card.denied {
  border-color: var(--el-color-danger);
  background: var(--el-color-danger-light-9);
}

.result-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
}

.success-icon {
  color: var(--el-color-success);
  font-size: 20px;
}

.error-icon {
  color: var(--el-color-danger);
  font-size: 20px;
}

.result-text {
  font-size: 18px;
  font-weight: bold;
}

.result-details {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.detail-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.detail-item label {
  width: 100px;
  font-weight: bold;
  color: var(--el-text-color-primary);
  flex-shrink: 0;
}

.policy-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.history-section {
  margin-top: 30px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.section-header h3 {
  margin: 0;
  color: var(--el-text-color-primary);
}
</style>