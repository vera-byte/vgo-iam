<template>
  <div class="developer-verification">
    <div class="page-header">
      <h1>开发者认证</h1>
      <el-button type="primary" @click="showCreateDialog = true">
        <el-icon>
          <Plus />
        </el-icon>
        新增认证
      </el-button>
    </div>

    <!-- 搜索和筛选 -->
    <el-card class="search-card" shadow="never">
      <el-form :model="searchForm" inline>
        <el-form-item label="开发者ID">
          <el-input v-model="searchForm.developerId" placeholder="请输入开发者ID" clearable />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="searchForm.status" placeholder="请选择状态" clearable>
            <el-option label="待审核" value="pending" />
            <el-option label="已通过" value="approved" />
            <el-option label="已拒绝" value="rejected" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadVerifications">搜索</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 认证列表 -->
    <el-card shadow="never">
      <el-table :data="verifications" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="developer_id" label="开发者ID" width="150" />
        <el-table-column prop="company_name" label="公司名称" width="200" />
        <el-table-column prop="contact_email" label="联系邮箱" width="200" />
        <el-table-column prop="status" label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">{{ getStatusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="申请时间" width="180">
          <template #default="{ row }">
            {{ dayjs(row.created_at).format('YYYY-MM-DD HH:mm:ss') }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="viewVerification(row)">查看</el-button>
            <el-button v-if="row.status === 'pending'" size="small" type="success"
              @click="approveVerification(row.id)">通过</el-button>
            <el-button v-if="row.status === 'pending'" size="small" type="danger"
              @click="rejectVerification(row.id)">拒绝</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination v-model:current-page="currentPage" v-model:page-size="pageSize" :total="total"
          :page-sizes="[10, 20, 50, 100]" layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadVerifications" @current-change="loadVerifications" />
      </div>
    </el-card>

    <!-- 创建认证对话框 -->
    <el-dialog v-model="showCreateDialog" title="新增开发者认证" width="600px">
      <el-form :model="newVerification" :rules="rules" ref="createFormRef" label-width="120px">
        <el-form-item label="开发者ID" prop="developer_id">
          <el-input v-model="newVerification.developer_id" placeholder="请输入开发者ID" />
        </el-form-item>
        <el-form-item label="公司名称" prop="company_name">
          <el-input v-model="newVerification.company_name" placeholder="请输入公司名称" />
        </el-form-item>
        <el-form-item label="联系邮箱" prop="contact_email">
          <el-input v-model="newVerification.contact_email" placeholder="请输入联系邮箱" />
        </el-form-item>
        <el-form-item label="联系电话" prop="contact_phone">
          <el-input v-model="newVerification.contact_phone" placeholder="请输入联系电话" />
        </el-form-item>
        <el-form-item label="业务描述" prop="business_description">
          <el-input v-model="newVerification.business_description" type="textarea" :rows="3" placeholder="请输入业务描述" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" @click="createVerification" :loading="creating">确定</el-button>
      </template>
    </el-dialog>

    <!-- 查看认证详情对话框 -->
    <el-dialog v-model="showDetailDialog" title="认证详情" width="600px">
      <el-descriptions :column="2" border v-if="currentVerification">
        <el-descriptions-item label="开发者ID">{{ currentVerification.developer_id }}</el-descriptions-item>
        <el-descriptions-item label="公司名称">{{ currentVerification.company_name }}</el-descriptions-item>
        <el-descriptions-item label="联系邮箱">{{ currentVerification.contact_email }}</el-descriptions-item>
        <el-descriptions-item label="联系电话">{{ currentVerification.contact_phone }}</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="getStatusType(currentVerification.status)">{{ getStatusText(currentVerification.status)
            }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="申请时间">{{ dayjs(currentVerification.created_at).format('YYYY-MM-DD HH:mm:ss')
          }}</el-descriptions-item>
        <el-descriptions-item label="业务描述" :span="2">{{ currentVerification.business_description
          }}</el-descriptions-item>
        <el-descriptions-item v-if="currentVerification.review_comment" label="审核意见" :span="2">{{
          currentVerification.review_comment }}</el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import dayjs from 'dayjs'
import { api } from '@/api'

interface DeveloperVerification {
  id: number
  developer_id: string
  company_name: string
  contact_email: string
  contact_phone: string
  business_description: string
  status: 'pending' | 'approved' | 'rejected'
  review_comment?: string
  created_at: string
  updated_at: string
}

const loading = ref(false)
const creating = ref(false)
const verifications = ref<DeveloperVerification[]>([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(10)

const searchForm = reactive({
  developerId: '',
  status: ''
})

const showCreateDialog = ref(false)
const showDetailDialog = ref(false)
const currentVerification = ref<DeveloperVerification | null>(null)
const createFormRef = ref()

const newVerification = reactive({
  developer_id: '',
  company_name: '',
  contact_email: '',
  contact_phone: '',
  business_description: ''
})

const rules = {
  developer_id: [{ required: true, message: '请输入开发者ID', trigger: 'blur' }],
  company_name: [{ required: true, message: '请输入公司名称', trigger: 'blur' }],
  contact_email: [
    { required: true, message: '请输入联系邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
  ],
  contact_phone: [{ required: true, message: '请输入联系电话', trigger: 'blur' }],
  business_description: [{ required: true, message: '请输入业务描述', trigger: 'blur' }]
}

const getStatusType = (status: string) => {
  const types: Record<string, string> = {
    pending: 'warning',
    approved: 'success',
    rejected: 'danger'
  }
  return types[status] || 'info'
}

const getStatusText = (status: string) => {
  const texts: Record<string, string> = {
    pending: '待审核',
    approved: '已通过',
    rejected: '已拒绝'
  }
  return texts[status] || status
}

const loadVerifications = async () => {
  loading.value = true
  try {
    const params = {
      page: currentPage.value,
      page_size: pageSize.value,
      developer_id: searchForm.developerId || undefined,
      status: searchForm.status || undefined
    }
    const response: any = await api.developerVerification.list(params)
    // 适配新的分页响应格式
    verifications.value = response.list || []
    total.value = response.pagination?.total || 0
  } catch (error) {
    ElMessage.error('加载认证列表失败')
  } finally {
    loading.value = false
  }
}

const resetSearch = () => {
  searchForm.developerId = ''
  searchForm.status = ''
  currentPage.value = 1
  loadVerifications()
}

const createVerification = async () => {
  if (!createFormRef.value) return

  try {
    await createFormRef.value.validate()
    creating.value = true

    await api.developerVerification.submit(newVerification)
    ElMessage.success('创建认证申请成功')
    showCreateDialog.value = false

    // 重置表单
    Object.assign(newVerification, {
      developer_id: '',
      company_name: '',
      contact_email: '',
      contact_phone: '',
      business_description: ''
    })

    loadVerifications()
  } catch (error) {
    ElMessage.error('创建认证申请失败')
  } finally {
    creating.value = false
  }
}

const viewVerification = (verification: DeveloperVerification) => {
  currentVerification.value = verification
  showDetailDialog.value = true
}

const approveVerification = async (id: number) => {
  try {
    await ElMessageBox.confirm('确定要通过这个认证申请吗？', '确认操作', {
      type: 'warning'
    })

    await api.developerVerification.review(id.toString(), { status: 'approved' })
    ElMessage.success('认证申请已通过')
    loadVerifications()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('操作失败')
    }
  }
}

const rejectVerification = async (id: number) => {
  try {
    const { value: comment } = await ElMessageBox.prompt('请输入拒绝原因', '拒绝认证', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      inputType: 'textarea',
      inputValidator: (value) => {
        if (!value) {
          return '请输入拒绝原因'
        }
        return true
      }
    })

    await api.developerVerification.review(id.toString(), { status: 'rejected', comment })
    ElMessage.success('认证申请已拒绝')
    loadVerifications()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('操作失败')
    }
  }
}

onMounted(() => {
  loadVerifications()
})
</script>

<style scoped>
.developer-verification {
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

.search-card {
  margin-bottom: 20px;
}

.pagination-wrapper {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}
</style>