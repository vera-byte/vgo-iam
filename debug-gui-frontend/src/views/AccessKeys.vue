<template>
  <div class="access-keys-container">
    <div class="page-header">
      <h1>访问密钥管理</h1>
      <el-button type="primary" @click="showCreateDialog = true">
        <el-icon>
          <Plus />
        </el-icon>
        创建访问密钥
      </el-button>
    </div>

    <div class="content-card">
      <div class="toolbar">
        <el-input v-model="searchQuery" placeholder="搜索访问密钥..." style="width: 300px" clearable>
          <template #prefix>
            <el-icon>
              <Search />
            </el-icon>
          </template>
        </el-input>
        <el-select v-model="statusFilter" placeholder="状态筛选" style="width: 120px">
          <el-option label="全部" value="" />
          <el-option label="活跃" value="active" />
          <el-option label="禁用" value="inactive" />
        </el-select>
      </div>

      <el-table :data="filteredAccessKeys" v-loading="loading" stripe>
        <el-table-column prop="access_key_id" label="访问密钥ID" width="200" />
        <el-table-column prop="user_id" label="用户ID" width="150" />
        <el-table-column prop="description" label="描述" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'danger'">
              {{ row.status === 'active' ? '活跃' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="expires_at" label="过期时间" width="180">
          <template #default="{ row }">
            {{ row.expires_at ? formatDate(row.expires_at) : '永不过期' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button size="small" @click="toggleStatus(row)">
              {{ row.status === 'active' ? '禁用' : '启用' }}
            </el-button>
            <el-button size="small" type="danger" @click="deleteAccessKey(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination v-model:current-page="currentPage" v-model:page-size="pageSize" :total="total"
          :page-sizes="[10, 20, 50, 100]" layout="total, sizes, prev, pager, next, jumper" @size-change="loadAccessKeys"
          @current-change="loadAccessKeys" />
      </div>
    </div>

    <!-- 创建访问密钥对话框 -->
    <el-dialog v-model="showCreateDialog" title="创建访问密钥" width="500px">
      <el-form :model="createForm" :rules="createRules" ref="createFormRef" label-width="100px">
        <el-form-item label="用户ID" prop="user_id">
          <el-input v-model="createForm.user_id" placeholder="请输入用户ID" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="createForm.description" type="textarea" placeholder="请输入访问密钥描述" :rows="3" />
        </el-form-item>
        <el-form-item label="过期时间">
          <el-date-picker v-model="createForm.expires_at" type="datetime" placeholder="选择过期时间（可选）"
            format="YYYY-MM-DD HH:mm:ss" value-format="YYYY-MM-DDTHH:mm:ssZ" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" @click="createAccessKey" :loading="creating">
          创建
        </el-button>
      </template>
    </el-dialog>

    <!-- 访问密钥详情对话框 -->
    <el-dialog v-model="showSecretDialog" title="访问密钥创建成功" width="600px">
      <el-alert title="请妥善保存以下信息" type="warning" description="密钥只会显示一次，请立即复制并妥善保存" show-icon :closable="false" />
      <div class="secret-info">
        <div class="secret-item">
          <label>访问密钥ID:</label>
          <div class="secret-value">
            <code>{{ newAccessKey.access_key_id }}</code>
            <el-button size="small" @click="copyToClipboard(newAccessKey.access_key_id)">
              复制
            </el-button>
          </div>
        </div>
        <div class="secret-item">
          <label>访问密钥:</label>
          <div class="secret-value">
            <code>{{ newAccessKey.secret_access_key }}</code>
            <el-button size="small" @click="copyToClipboard(newAccessKey.secret_access_key)">
              复制
            </el-button>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button type="primary" @click="showSecretDialog = false">
          我已保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Search } from '@element-plus/icons-vue'
import { api } from '../api'
import * as dayjs from 'dayjs'

interface AccessKey {
  access_key_id: string
  user_id: string
  description: string
  status: 'active' | 'inactive'
  created_at: string
  expires_at?: string
}

interface CreateAccessKeyForm {
  user_id: string
  description: string
  expires_at?: string
}

const loading = ref(false)
const creating = ref(false)
const accessKeys = ref<AccessKey[]>([])
const searchQuery = ref('')
const statusFilter = ref('')
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

const showCreateDialog = ref(false)
const showSecretDialog = ref(false)
const createFormRef = ref()
const createForm = ref<CreateAccessKeyForm>({
  user_id: '',
  description: ''
})

const newAccessKey = ref({
  access_key_id: '',
  secret_access_key: ''
})

const createRules = {
  user_id: [{ required: true, message: '请输入用户ID', trigger: 'blur' }],
  description: [{ required: true, message: '请输入描述', trigger: 'blur' }]
}

const filteredAccessKeys = computed(() => {
  let filtered = accessKeys.value

  if (searchQuery.value) {
    filtered = filtered.filter(key =>
      key.access_key_id.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
      key.user_id.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
      key.description.toLowerCase().includes(searchQuery.value.toLowerCase())
    )
  }

  if (statusFilter.value) {
    filtered = filtered.filter(key => key.status === statusFilter.value)
  }

  return filtered
})

const formatDate = (dateString: string) => {
  return dayjs.default(dateString).format('YYYY-MM-DD HH:mm:ss')
}

const loadAccessKeys = async () => {
  loading.value = true
  try {
    const response: any = await api.accessKeys.list({
      page: currentPage.value,
      page_size: pageSize.value,
      status: statusFilter.value || undefined
    })
    // 适配新的分页响应格式
    accessKeys.value = response.list || []
    total.value = response.pagination?.total || 0
  } catch (error) {
    console.error('加载访问密钥失败:', error)
    ElMessage.error('加载访问密钥失败')
  } finally {
    loading.value = false
  }
}

const createAccessKey = async () => {
  if (!createFormRef.value) return

  try {
    await createFormRef.value.validate()
    creating.value = true

    const response = await api.accessKeys.create(createForm.value)
    newAccessKey.value = response.data

    showCreateDialog.value = false
    showSecretDialog.value = true

    // 重置表单
    createForm.value = {
      user_id: '',
      description: ''
    }

    // 重新加载列表
    await loadAccessKeys()

    ElMessage.success('访问密钥创建成功')
  } catch (error) {
    console.error('创建访问密钥失败:', error)
    ElMessage.error('创建访问密钥失败')
  } finally {
    creating.value = false
  }
}

const toggleStatus = async (accessKey: AccessKey) => {
  try {
    const newStatus = accessKey.status === 'active' ? 'inactive' : 'active'
    await api.accessKeys.updateStatus(accessKey.access_key_id, newStatus)
    accessKey.status = newStatus
    ElMessage.success(`访问密钥已${newStatus === 'active' ? '启用' : '禁用'}`)
  } catch (error) {
    console.error('更新访问密钥状态失败:', error)
    ElMessage.error('更新访问密钥状态失败')
  }
}

const deleteAccessKey = async (accessKey: AccessKey) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除访问密钥 ${accessKey.access_key_id} 吗？`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    await api.accessKeys.updateStatus(accessKey.access_key_id, 'deleted')
    await loadAccessKeys()
    ElMessage.success('访问密钥删除成功')
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除访问密钥失败:', error)
      ElMessage.error('删除访问密钥失败')
    }
  }
}

const copyToClipboard = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制到剪贴板')
  } catch (error) {
    console.error('复制失败:', error)
    ElMessage.error('复制失败')
  }
}

onMounted(() => {
  loadAccessKeys()
})
</script>

<style scoped>
.access-keys-container {
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

.toolbar {
  display: flex;
  gap: 16px;
  margin-bottom: 20px;
}

.pagination {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}

.secret-info {
  margin: 20px 0;
}

.secret-item {
  margin-bottom: 16px;
}

.secret-item label {
  display: block;
  margin-bottom: 8px;
  font-weight: bold;
  color: var(--el-text-color-primary);
}

.secret-value {
  display: flex;
  align-items: center;
  gap: 12px;
}

.secret-value code {
  flex: 1;
  padding: 8px 12px;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  word-break: break-all;
}
</style>