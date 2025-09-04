<template>
  <div class="applications-container">
    <div class="page-header">
      <h1>应用管理</h1>
      <el-button type="primary" @click="showCreateDialog = true">
        <el-icon><Plus /></el-icon>
        创建应用
      </el-button>
    </div>

    <div class="content-card">
      <div class="toolbar">
        <el-input
          v-model="searchQuery"
          placeholder="搜索应用..."
          style="width: 300px"
          clearable
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-select v-model="typeFilter" placeholder="应用类型" style="width: 120px">
          <el-option label="全部" value="" />
          <el-option label="Web应用" value="web" />
          <el-option label="移动应用" value="mobile" />
          <el-option label="桌面应用" value="desktop" />
        </el-select>
      </div>

      <el-table :data="filteredApplications" v-loading="loading" stripe>
        <el-table-column prop="id" label="应用ID" width="200" />
        <el-table-column prop="name" label="应用名称" width="200" />
        <el-table-column prop="type" label="类型" width="100">
          <template #default="{ row }">
            <el-tag :type="getTypeTagType(row.type)">{{ getTypeLabel(row.type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" />
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button size="small" @click="viewApplication(row)">查看</el-button>
            <el-button size="small" @click="editApplication(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="deleteApplication(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadApplications"
          @current-change="loadApplications"
        />
      </div>
    </div>

    <!-- 创建/编辑应用对话框 -->
    <el-dialog 
      v-model="showCreateDialog" 
      :title="editingApplication ? '编辑应用' : '创建应用'" 
      width="600px"
    >
      <el-form :model="applicationForm" :rules="applicationRules" ref="applicationFormRef" label-width="120px">
        <el-form-item label="应用名称" prop="name">
          <el-input v-model="applicationForm.name" placeholder="请输入应用名称" />
        </el-form-item>
        <el-form-item label="应用类型" prop="type">
          <el-select v-model="applicationForm.type" placeholder="请选择应用类型">
            <el-option label="Web应用" value="web" />
            <el-option label="移动应用" value="mobile" />
            <el-option label="桌面应用" value="desktop" />
          </el-select>
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input
            v-model="applicationForm.description"
            type="textarea"
            placeholder="请输入应用描述"
            :rows="3"
          />
        </el-form-item>
        <el-form-item label="重定向URI">
          <div class="redirect-uris">
            <div v-for="(uri, index) in applicationForm.redirect_uris" :key="index" class="uri-item">
              <el-input v-model="applicationForm.redirect_uris[index]" placeholder="请输入重定向URI" />
              <el-button 
                type="danger" 
                size="small" 
                @click="removeRedirectUri(index)"
                :disabled="applicationForm.redirect_uris.length <= 1"
              >
                删除
              </el-button>
            </div>
            <el-button type="primary" size="small" @click="addRedirectUri">
              添加URI
            </el-button>
          </div>
        </el-form-item>
        <el-form-item label="权限范围">
          <el-checkbox-group v-model="applicationForm.scopes">
            <el-checkbox label="read">读取</el-checkbox>
            <el-checkbox label="write">写入</el-checkbox>
            <el-checkbox label="admin">管理</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="cancelEdit">取消</el-button>
        <el-button type="primary" @click="saveApplication" :loading="saving">
          {{ editingApplication ? '更新' : '创建' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 查看应用对话框 -->
    <el-dialog v-model="showViewDialog" title="应用详情" width="600px">
      <div v-if="viewingApplication">
        <div class="app-info">
          <div class="info-item">
            <label>应用ID:</label>
            <span>{{ viewingApplication.id }}</span>
          </div>
          <div class="info-item">
            <label>应用名称:</label>
            <span>{{ viewingApplication.name }}</span>
          </div>
          <div class="info-item">
            <label>应用类型:</label>
            <el-tag :type="getTypeTagType(viewingApplication.type)">
              {{ getTypeLabel(viewingApplication.type) }}
            </el-tag>
          </div>
          <div class="info-item">
            <label>描述:</label>
            <span>{{ viewingApplication.description }}</span>
          </div>
          <div class="info-item">
            <label>重定向URI:</label>
            <div class="uri-list">
              <div v-for="uri in viewingApplication.redirect_uris" :key="uri" class="uri-tag">
                <el-tag>{{ uri }}</el-tag>
              </div>
            </div>
          </div>
          <div class="info-item">
            <label>权限范围:</label>
            <div class="scope-list">
              <el-tag v-for="scope in viewingApplication.scopes" :key="scope" type="info">
                {{ getScopeLabel(scope) }}
              </el-tag>
            </div>
          </div>
          <div class="info-item">
            <label>创建时间:</label>
            <span>{{ formatDate(viewingApplication.created_at) }}</span>
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Search } from '@element-plus/icons-vue'
import { api } from '../api'
import * as dayjs from 'dayjs'

interface Application {
  id: string
  name: string
  type: 'web' | 'mobile' | 'desktop'
  description: string
  redirect_uris: string[]
  scopes: string[]
  created_at: string
  updated_at: string
}

interface ApplicationForm {
  name: string
  type: 'web' | 'mobile' | 'desktop' | ''
  description: string
  redirect_uris: string[]
  scopes: string[]
}

const loading = ref(false)
const saving = ref(false)
const applications = ref<Application[]>([])
const searchQuery = ref('')
const typeFilter = ref('')
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

const showCreateDialog = ref(false)
const showViewDialog = ref(false)
const editingApplication = ref<Application | null>(null)
const viewingApplication = ref<Application | null>(null)
const applicationFormRef = ref()

const applicationForm = ref<ApplicationForm>({
  name: '',
  type: '',
  description: '',
  redirect_uris: [''],
  scopes: []
})

const applicationRules = {
  name: [{ required: true, message: '请输入应用名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择应用类型', trigger: 'change' }],
  description: [{ required: true, message: '请输入描述', trigger: 'blur' }]
}

const filteredApplications = computed(() => {
  let filtered = applications.value
  
  if (searchQuery.value) {
    filtered = filtered.filter(app => 
      app.name.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
      app.description.toLowerCase().includes(searchQuery.value.toLowerCase())
    )
  }
  
  if (typeFilter.value) {
    filtered = filtered.filter(app => app.type === typeFilter.value)
  }
  
  return filtered
})

const getTypeLabel = (type: string) => {
  const labels: Record<string, string> = {
    web: 'Web应用',
    mobile: '移动应用',
    desktop: '桌面应用'
  }
  return labels[type] || type
}

const getTypeTagType = (type: string) => {
  const types: Record<string, string> = {
    web: 'primary',
    mobile: 'success',
    desktop: 'warning'
  }
  return types[type] || 'info'
}

const getScopeLabel = (scope: string) => {
  const labels: Record<string, string> = {
    read: '读取',
    write: '写入',
    admin: '管理'
  }
  return labels[scope] || scope
}

const formatDate = (dateString: string) => {
  return dayjs(dateString).format('YYYY-MM-DD HH:mm:ss')
}

const loadApplications = async () => {
  loading.value = true
  try {
    const response = await api.applications.list({
      page: currentPage.value,
      page_size: pageSize.value,
      type: typeFilter.value || undefined
    })
    applications.value = response.data.applications || []
    total.value = response.data.total || 0
  } catch (error) {
    console.error('加载应用失败:', error)
    ElMessage.error('加载应用失败')
  } finally {
    loading.value = false
  }
}

const viewApplication = (application: Application) => {
  viewingApplication.value = application
  showViewDialog.value = true
}

const editApplication = (application: Application) => {
  editingApplication.value = application
  applicationForm.value = {
    name: application.name,
    type: application.type,
    description: application.description,
    redirect_uris: [...application.redirect_uris],
    scopes: [...application.scopes]
  }
  showCreateDialog.value = true
}

const cancelEdit = () => {
  showCreateDialog.value = false
  editingApplication.value = null
  applicationForm.value = {
    name: '',
    type: '',
    description: '',
    redirect_uris: [''],
    scopes: []
  }
}

const addRedirectUri = () => {
  applicationForm.value.redirect_uris.push('')
}

const removeRedirectUri = (index: number) => {
  applicationForm.value.redirect_uris.splice(index, 1)
}

const saveApplication = async () => {
  if (!applicationFormRef.value) return
  
  try {
    await applicationFormRef.value.validate()
    
    // 过滤空的重定向URI
    const formData = {
      ...applicationForm.value,
      redirect_uris: applicationForm.value.redirect_uris.filter(uri => uri.trim())
    }
    
    saving.value = true
    
    if (editingApplication.value) {
      // 更新应用
      await api.applications.update(editingApplication.value.id, formData)
      ElMessage.success('应用更新成功')
    } else {
      // 创建应用
      await api.applications.create(formData)
      ElMessage.success('应用创建成功')
    }
    
    cancelEdit()
    await loadApplications()
  } catch (error) {
    console.error('保存应用失败:', error)
    ElMessage.error('保存应用失败')
  } finally {
    saving.value = false
  }
}

const deleteApplication = async (application: Application) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除应用 "${application.name}" 吗？`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    await api.applications.delete(application.id)
    await loadApplications()
    ElMessage.success('应用删除成功')
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除应用失败:', error)
      ElMessage.error('删除应用失败')
    }
  }
}

onMounted(() => {
  loadApplications()
})
</script>

<style scoped>
.applications-container {
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

.redirect-uris {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.uri-item {
  display: flex;
  gap: 8px;
  align-items: center;
}

.app-info {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.info-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.info-item label {
  width: 100px;
  font-weight: bold;
  color: var(--el-text-color-primary);
  flex-shrink: 0;
}

.uri-list,
.scope-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.uri-tag {
  margin-bottom: 4px;
}
</style>