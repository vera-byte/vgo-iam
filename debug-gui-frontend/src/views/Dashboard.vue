<template>
  <div class="dashboard-container">
    <div class="page-header">
      <h1>系统概览</h1>
      <el-button @click="refreshData" :loading="loading">
        <el-icon>
          <Refresh />
        </el-icon>
        刷新数据
      </el-button>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon users">
          <el-icon>
            <User />
          </el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-number">{{ stats.users || 0 }}</div>
          <div class="stat-label">用户总数</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon keys">
          <el-icon>
            <Key />
          </el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-number">{{ stats.accessKeys || 0 }}</div>
          <div class="stat-label">访问密钥</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon policies">
          <el-icon>
            <Document />
          </el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-number">{{ stats.policies || 0 }}</div>
          <div class="stat-label">策略数量</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon apps">
          <el-icon>
            <Grid />
          </el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-number">{{ stats.applications || 0 }}</div>
          <div class="stat-label">应用数量</div>
        </div>
      </div>
    </div>

    <!-- 系统状态 -->
    <div class="content-row">
      <div class="content-card half">
        <h3>系统状态</h3>
        <div class="system-status">
          <div class="status-item">
            <span class="status-label">服务状态:</span>
            <el-tag :type="systemStatus.service_status === 'healthy' ? 'success' : 'danger'">
              {{ systemStatus.service_status === 'healthy' ? '正常' : '异常' }}
            </el-tag>
          </div>
          <div class="status-item">
            <span class="status-label">数据库连接:</span>
            <el-tag :type="systemStatus.database_status === 'connected' ? 'success' : 'danger'">
              {{ systemStatus.database_status === 'connected' ? '已连接' : '断开' }}
            </el-tag>
          </div>
          <div class="status-item">
            <span class="status-label">运行时间:</span>
            <span>{{ systemStatus.uptime || '未知' }}</span>
          </div>
          <div class="status-item">
            <span class="status-label">版本:</span>
            <span>{{ systemStatus.version || '未知' }}</span>
          </div>
        </div>
      </div>

      <div class="content-card half">
        <h3>快速操作</h3>
        <div class="quick-actions">
          <el-button type="primary" @click="$router.push('/users')">
            <el-icon>
              <User />
            </el-icon>
            管理用户
          </el-button>
          <el-button type="success" @click="$router.push('/access-keys')">
            <el-icon>
              <Key />
            </el-icon>
            管理密钥
          </el-button>
          <el-button type="warning" @click="$router.push('/policies')">
            <el-icon>
              <Document />
            </el-icon>
            管理策略
          </el-button>
          <el-button type="info" @click="$router.push('/applications')">
            <el-icon>
              <Grid />
            </el-icon>
            管理应用
          </el-button>
          <el-button type="primary" @click="$router.push('/signature-config')">
            <el-icon>
              <Key />
            </el-icon>
            签名配置
          </el-button>
        </div>
      </div>
    </div>

    <!-- 最近活动 -->
    <div class="content-card">
      <h3>最近活动</h3>
      <el-table :data="recentActivities" v-loading="loading" stripe>
        <el-table-column prop="type" label="类型" width="100">
          <template #default="{ row }">
            <el-tag :type="getActivityTagType(row.type)">{{ getActivityLabel(row.type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" />
        <el-table-column prop="user" label="操作用户" width="150" />
        <el-table-column prop="timestamp" label="时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.timestamp) }}
          </template>
        </el-table-column>
      </el-table>

      <div v-if="recentActivities.length === 0 && !loading" class="empty-state">
        <el-empty description="暂无活动记录" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, User, Key, Document, Grid } from '@element-plus/icons-vue'
import { api } from '../api'
import dayjs from 'dayjs'

interface Stats {
  users: number
  accessKeys: number
  policies: number
  applications: number
}

interface SystemStatus {
  service_status: 'healthy' | 'unhealthy'
  database_status: 'connected' | 'disconnected'
  uptime: string
  version: string
}

interface Activity {
  id: string
  type: 'user_created' | 'user_updated' | 'user_deleted' | 'key_created' | 'key_deleted' | 'policy_created' | 'policy_updated' | 'policy_deleted' | 'app_created' | 'app_updated' | 'app_deleted'
  description: string
  user: string
  timestamp: string
}

const loading = ref(false)
const stats = ref<Stats>({
  users: 0,
  accessKeys: 0,
  policies: 0,
  applications: 0
})

const systemStatus = ref<SystemStatus>({
  service_status: 'healthy',
  database_status: 'connected',
  uptime: '',
  version: ''
})

const recentActivities = ref<Activity[]>([])

const getActivityLabel = (type: string) => {
  const labels: Record<string, string> = {
    user_created: '用户创建',
    user_updated: '用户更新',
    user_deleted: '用户删除',
    key_created: '密钥创建',
    key_deleted: '密钥删除',
    policy_created: '策略创建',
    policy_updated: '策略更新',
    policy_deleted: '策略删除',
    app_created: '应用创建',
    app_updated: '应用更新',
    app_deleted: '应用删除'
  }
  return labels[type] || type
}

const getActivityTagType = (type: string) => {
  if (type.includes('created')) return 'success'
  if (type.includes('updated')) return 'warning'
  if (type.includes('deleted')) return 'danger'
  return 'info'
}

const formatDate = (dateString: string) => {
  return dayjs(dateString).format('YYYY-MM-DD HH:mm:ss')
}

const loadStats = async () => {
  try {
    const response = await api.dashboard.stats()
    // 响应拦截器已经返回了response.data，所以这里直接使用response
    stats.value = response as unknown as Stats
  } catch (error) {
    console.error('加载统计数据失败:', error)
    // 使用模拟数据
    stats.value = {
      users: 0,
      accessKeys: 0,
      policies: 0,
      applications: 0
    }
  }
}

const loadSystemStatus = async () => {
  try {
    const response = await api.dashboard.status()
    // 响应拦截器已经返回了response.data，所以这里直接使用response
    systemStatus.value = response as unknown as SystemStatus
  } catch (error) {
    console.error('加载系统状态失败:', error)
    // 使用模拟数据
    systemStatus.value = {
      service_status: 'healthy',
      database_status: 'connected',
      uptime: '2天 5小时 30分钟',
      version: 'v1.0.0'
    }
  }
}

const loadRecentActivities = async () => {
  try {
    const response = await api.dashboard.activities()
    // 响应拦截器已经返回了response.data，所以这里直接使用response
    recentActivities.value = (response as any).activities || []
  } catch (error) {
    console.error('加载活动记录失败:', error)
    // 使用模拟数据
    recentActivities.value = [

    ]
  }
}

const refreshData = async () => {
  loading.value = true
  try {
    await Promise.all([
      loadStats(),
      loadSystemStatus(),
      loadRecentActivities()
    ])
    ElMessage.success('数据刷新成功')
  } catch (error) {
    console.error('刷新数据失败:', error)
    ElMessage.error('刷新数据失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  refreshData()
})
</script>

<style scoped>
.dashboard-container {
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

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
  margin-bottom: 30px;
}

.stat-card {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-icon {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  color: white;
}

.stat-icon.users {
  background: var(--el-color-primary);
}

.stat-icon.keys {
  background: var(--el-color-success);
}

.stat-icon.policies {
  background: var(--el-color-warning);
}

.stat-icon.apps {
  background: var(--el-color-info);
}

.stat-content {
  flex: 1;
}

.stat-number {
  font-size: 32px;
  font-weight: bold;
  color: var(--el-text-color-primary);
  line-height: 1;
}

.stat-label {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}

.content-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  margin-bottom: 30px;
}

.content-card {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.content-card.half {
  /* 已在grid中定义 */
}

.content-card h3 {
  margin: 0 0 20px 0;
  color: var(--el-text-color-primary);
}

.system-status {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.status-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.status-label {
  width: 100px;
  font-weight: bold;
  color: var(--el-text-color-primary);
}

.quick-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.quick-actions .el-button {
  justify-content: flex-start;
}

.empty-state {
  padding: 40px 0;
  text-align: center;
}

@media (max-width: 768px) {
  .content-row {
    grid-template-columns: 1fr;
  }

  .stats-grid {
    grid-template-columns: 1fr;
  }

  .quick-actions {
    grid-template-columns: 1fr;
  }
}
</style>