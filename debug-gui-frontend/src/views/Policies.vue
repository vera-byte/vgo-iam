<template>
  <div class="policies-container">
    <div class="page-header">
      <h1>策略管理</h1>
      <el-button type="primary" @click="showCreateDialog = true">
        <el-icon>
          <Plus />
        </el-icon>
        创建策略
      </el-button>
    </div>

    <div class="content-card">
      <div class="toolbar">
        <el-input v-model="searchQuery" placeholder="搜索策略..." style="width: 300px" clearable>
          <template #prefix>
            <el-icon>
              <Search />
            </el-icon>
          </template>
        </el-input>
      </div>

      <el-table :data="filteredPolicies" v-loading="loading" stripe>
        <el-table-column prop="id" label="策略ID" width="200" />
        <el-table-column prop="name" label="策略名称" width="200" />
        <el-table-column prop="description" label="描述" />
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button size="small" @click="viewPolicy(row)">查看</el-button>
            <el-button size="small" @click="editPolicy(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="deletePolicy(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination v-model:current-page="currentPage" v-model:page-size="pageSize" :total="total"
          :page-sizes="[10, 20, 50, 100]" layout="total, sizes, prev, pager, next, jumper" @size-change="loadPolicies"
          @current-change="loadPolicies" />
      </div>
    </div>

    <!-- 创建/编辑策略对话框 -->
    <el-dialog v-model="showCreateDialog" :title="editingPolicy ? '编辑策略' : '创建策略'" width="800px">
      <el-form :model="policyForm" :rules="policyRules" ref="policyFormRef" label-width="100px">
        <el-form-item label="策略名称" prop="name">
          <el-input v-model="policyForm.name" placeholder="请输入策略名称" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="policyForm.description" type="textarea" placeholder="请输入策略描述" :rows="3" />
        </el-form-item>
        <el-form-item label="策略文档" prop="policy_document">
          <el-input v-model="policyDocumentText" type="textarea" placeholder="请输入JSON格式的策略文档" :rows="10"
            style="font-family: monospace" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="cancelEdit">取消</el-button>
        <el-button type="primary" @click="savePolicy" :loading="saving">
          {{ editingPolicy ? '更新' : '创建' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 查看策略对话框 -->
    <el-dialog v-model="showViewDialog" title="策略详情" width="800px">
      <div v-if="viewingPolicy">
        <div class="policy-info">
          <div class="info-item">
            <label>策略ID:</label>
            <span>{{ viewingPolicy.id }}</span>
          </div>
          <div class="info-item">
            <label>策略名称:</label>
            <span>{{ viewingPolicy.name }}</span>
          </div>
          <div class="info-item">
            <label>描述:</label>
            <span>{{ viewingPolicy.description }}</span>
          </div>
          <div class="info-item">
            <label>创建时间:</label>
            <span>{{ formatDate(viewingPolicy.created_at) }}</span>
          </div>
        </div>
        <div class="policy-document">
          <label>策略文档:</label>
          <pre><code>{{ JSON.stringify(viewingPolicy.policy_document, null, 2) }}</code></pre>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Search } from '@element-plus/icons-vue'
import { api } from '../api'
import dayjs from 'dayjs'

interface Policy {
  id: string
  name: string
  description: string
  policy_document: any
  created_at: string
  updated_at: string
}

interface PolicyForm {
  name: string
  description: string
  policy_document: any
}

const loading = ref(false)
const saving = ref(false)
const policies = ref<Policy[]>([])
const searchQuery = ref('')
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

const showCreateDialog = ref(false)
const showViewDialog = ref(false)
const editingPolicy = ref<Policy | null>(null)
const viewingPolicy = ref<Policy | null>(null)
const policyFormRef = ref()

const policyForm = ref<PolicyForm>({
  name: '',
  description: '',
  policy_document: {}
})

const policyDocumentText = ref('')

const policyRules = {
  name: [{ required: true, message: '请输入策略名称', trigger: 'blur' }],
  description: [{ required: true, message: '请输入描述', trigger: 'blur' }],
  policy_document: [{ required: true, message: '请输入策略文档', trigger: 'blur' }]
}

const filteredPolicies = computed(() => {
  if (!searchQuery.value) return policies.value

  return policies.value.filter(policy =>
    policy.name.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
    policy.description.toLowerCase().includes(searchQuery.value.toLowerCase())
  )
})

// 监听策略文档文本变化，解析JSON
watch(policyDocumentText, (newValue) => {
  try {
    if (newValue.trim()) {
      policyForm.value.policy_document = JSON.parse(newValue)
    } else {
      policyForm.value.policy_document = {}
    }
  } catch (error) {
    // JSON解析错误，保持原值
  }
})

const formatDate = (dateString: string) => {
  return dayjs(dateString).format('YYYY-MM-DD HH:mm:ss')
}

const loadPolicies = async () => {
  loading.value = true
  try {
    const response: any = await api.policies.list({
      page: currentPage.value,
      page_size: pageSize.value
    })
    // 适配新的分页响应格式
    policies.value = response.list || []
    total.value = response.pagination?.total || 0
  } catch (error) {
    console.error('加载策略失败:', error)
    ElMessage.error('加载策略失败')
  } finally {
    loading.value = false
  }
}

const viewPolicy = (policy: Policy) => {
  viewingPolicy.value = policy
  showViewDialog.value = true
}

const editPolicy = (policy: Policy) => {
  editingPolicy.value = policy
  policyForm.value = {
    name: policy.name,
    description: policy.description,
    policy_document: policy.policy_document
  }
  policyDocumentText.value = JSON.stringify(policy.policy_document, null, 2)
  showCreateDialog.value = true
}

const cancelEdit = () => {
  showCreateDialog.value = false
  editingPolicy.value = null
  policyForm.value = {
    name: '',
    description: '',
    policy_document: {}
  }
  policyDocumentText.value = ''
}

const savePolicy = async () => {
  if (!policyFormRef.value) return

  try {
    await policyFormRef.value.validate()

    // 验证JSON格式
    try {
      JSON.parse(policyDocumentText.value)
    } catch (error) {
      ElMessage.error('策略文档格式错误，请输入有效的JSON')
      return
    }

    saving.value = true

    if (editingPolicy.value) {
      // 更新策略
      await api.policies.update(editingPolicy.value.id, policyForm.value)
      ElMessage.success('策略更新成功')
    } else {
      // 创建策略
      await api.policies.create(policyForm.value)
      ElMessage.success('策略创建成功')
    }

    cancelEdit()
    await loadPolicies()
  } catch (error) {
    console.error('保存策略失败:', error)
    ElMessage.error('保存策略失败')
  } finally {
    saving.value = false
  }
}

const deletePolicy = async (policy: Policy) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除策略 "${policy.name}" 吗？`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    await api.policies.delete(policy.id)
    await loadPolicies()
    ElMessage.success('策略删除成功')
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除策略失败:', error)
      ElMessage.error('删除策略失败')
    }
  }
}

onMounted(() => {
  loadPolicies()
})
</script>

<style scoped>
.policies-container {
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

.policy-info {
  margin-bottom: 20px;
}

.info-item {
  display: flex;
  margin-bottom: 12px;
}

.info-item label {
  width: 100px;
  font-weight: bold;
  color: var(--el-text-color-primary);
}

.policy-document {
  margin-top: 20px;
}

.policy-document label {
  display: block;
  margin-bottom: 8px;
  font-weight: bold;
  color: var(--el-text-color-primary);
}

.policy-document pre {
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  padding: 16px;
  overflow-x: auto;
  max-height: 400px;
}

.policy-document code {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 14px;
  line-height: 1.5;
}
</style>