<template>
  <div class="users-page">
    <div class="toolbar">
      <div class="toolbar__left">
        <h2>用户管理</h2>
      </div>
      <div class="toolbar__right">
        <el-button type="primary" @click="showCreateDialog = true">
          创建用户
        </el-button>
        <el-button @click="loadUsers">
          刷新
        </el-button>
      </div>
    </div>

    <div class="table-container">
      <el-table :data="users" v-loading="loading" style="width: 100%">
        <el-table-column prop="id" label="ID" width="200" />
        <el-table-column prop="name" label="用户名" />
        <el-table-column prop="email" label="邮箱" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'danger'">
              {{ row.status === 'active' ? '激活' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="180">
          <template #default="{ row }">
            <span>{{ dayjs(row.createdAt).format("YYYY-MM-DD HH:mm:s") }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button size="small" @click="editUser(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="deleteUser(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 创建用户对话框 -->
    <el-dialog v-model="showCreateDialog" title="创建用户" width="500px">
      <el-form :model="userForm" label-width="80px">
        <el-form-item label="用户名">
          <el-input v-model="userForm.username" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="userForm.email" placeholder="请输入邮箱" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="userForm.password" type="password" placeholder="请输入密码" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button @click="createUser" type="primary" :loading="creating">
          创建
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api, type User } from '../api'
import dayjs from 'dayjs'

// 响应式数据
const users = ref<User[]>([])
const loading = ref(false)
const creating = ref(false)
const showCreateDialog = ref(false)

// 用户表单
const userForm = ref({
  username: '',
  email: '',
  password: ''
})

// 方法
const loadUsers = async () => {
  loading.value = true
  try {
    const response: any = await api.users.list()
    // 适配新的分页响应格式
    users.value = response.list || []
  } catch (error: any) {
    ElMessage.error(`加载用户列表失败: ${error.message}`)
  } finally {
    loading.value = false
  }
}

const createUser = async () => {
  if (!userForm.value.username || !userForm.value.email || !userForm.value.password) {
    ElMessage.error('请填写完整信息')
    return
  }

  creating.value = true
  try {
    await api.users.create(userForm.value)
    ElMessage.success('用户创建成功')
    showCreateDialog.value = false
    userForm.value = { username: '', email: '', password: '' }
    await loadUsers()
  } catch (error: any) {
    ElMessage.error(`创建用户失败: ${error.message}`)
  } finally {
    creating.value = false
  }
}

const editUser = (user: User) => {
  ElMessage.info('编辑功能待实现')
}

const deleteUser = async (id: string) => {
  try {
    await ElMessageBox.confirm('确定要删除这个用户吗？', '确认删除', {
      type: 'warning'
    })

    await api.users.delete(id)
    ElMessage.success('用户删除成功')
    await loadUsers()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(`删除用户失败: ${error.message}`)
    }
  }
}

// 生命周期
onMounted(() => {
  loadUsers()
})
</script>

<style lang="scss" scoped>
.users-page {
  padding: 20px;
}
</style>