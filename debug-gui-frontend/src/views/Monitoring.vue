<template>
  <div class="monitoring">
    <div class="page-header">
      <h1>系统监控</h1>
      <el-button @click="refreshData" :loading="loading">
        <el-icon><Refresh /></el-icon>
        刷新数据
      </el-button>
    </div>

    <!-- 系统状态概览 -->
    <el-row :gutter="20" class="status-cards">
      <el-col :span="6">
        <el-card class="status-card">
          <div class="status-item">
            <div class="status-icon healthy">
              <el-icon><CircleCheck /></el-icon>
            </div>
            <div class="status-content">
              <div class="status-title">服务状态</div>
              <div class="status-value">{{ systemStatus.service_status || '正常' }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="status-card">
          <div class="status-item">
            <div class="status-icon">
              <el-icon><Connection /></el-icon>
            </div>
            <div class="status-content">
              <div class="status-title">数据库连接</div>
              <div class="status-value">{{ systemStatus.database_status || '正常' }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="status-card">
          <div class="status-item">
            <div class="status-icon">
              <el-icon><Timer /></el-icon>
            </div>
            <div class="status-content">
              <div class="status-title">运行时间</div>
              <div class="status-value">{{ formatUptime(systemStatus.uptime) }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="status-card">
          <div class="status-item">
            <div class="status-icon">
              <el-icon><Monitor /></el-icon>
            </div>
            <div class="status-content">
              <div class="status-title">版本</div>
              <div class="status-value">{{ systemStatus.version || 'v1.0.0' }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 性能指标 -->
    <el-row :gutter="20" class="metrics-section">
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>系统资源使用情况</span>
            </div>
          </template>
          <div class="metrics-grid">
            <div class="metric-item">
              <div class="metric-label">CPU 使用率</div>
              <el-progress :percentage="metrics.cpu_usage || 0" :color="getProgressColor(metrics.cpu_usage)" />
            </div>
            <div class="metric-item">
              <div class="metric-label">内存使用率</div>
              <el-progress :percentage="metrics.memory_usage || 0" :color="getProgressColor(metrics.memory_usage)" />
            </div>
            <div class="metric-item">
              <div class="metric-label">磁盘使用率</div>
              <el-progress :percentage="metrics.disk_usage || 0" :color="getProgressColor(metrics.disk_usage)" />
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>API 请求统计</span>
            </div>
          </template>
          <div class="api-stats">
            <div class="stat-row">
              <span class="stat-label">总请求数:</span>
              <span class="stat-value">{{ metrics.total_requests || 0 }}</span>
            </div>
            <div class="stat-row">
              <span class="stat-label">成功请求:</span>
              <span class="stat-value success">{{ metrics.successful_requests || 0 }}</span>
            </div>
            <div class="stat-row">
              <span class="stat-label">失败请求:</span>
              <span class="stat-value error">{{ metrics.failed_requests || 0 }}</span>
            </div>
            <div class="stat-row">
              <span class="stat-label">平均响应时间:</span>
              <span class="stat-value">{{ metrics.avg_response_time || 0 }}ms</span>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 实时日志 -->
    <el-card class="logs-section">
      <template #header>
        <div class="card-header">
          <span>实时日志</span>
          <div class="log-controls">
            <el-select v-model="logLevel" placeholder="日志级别" style="width: 120px; margin-right: 10px;">
              <el-option label="全部" value="" />
              <el-option label="ERROR" value="error" />
              <el-option label="WARN" value="warn" />
              <el-option label="INFO" value="info" />
              <el-option label="DEBUG" value="debug" />
            </el-select>
            <el-button @click="clearLogs" size="small">清空日志</el-button>
            <el-button @click="toggleAutoRefresh" size="small" :type="autoRefresh ? 'success' : 'default'">
              {{ autoRefresh ? '停止自动刷新' : '开启自动刷新' }}
            </el-button>
          </div>
        </div>
      </template>
      <div class="logs-container" ref="logsContainer">
        <div v-if="filteredLogs.length === 0" class="empty-logs">
          暂无日志数据
        </div>
        <div v-else>
          <div v-for="log in filteredLogs" :key="log.id" class="log-entry" :class="log.level">
            <span class="log-time">{{ dayjs(log.timestamp).format('HH:mm:ss') }}</span>
            <span class="log-level">{{ log.level.toUpperCase() }}</span>
            <span class="log-message">{{ log.message }}</span>
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, CircleCheck, Connection, Timer, Monitor } from '@element-plus/icons-vue'
import * as dayjs from 'dayjs'
import { api } from '@/api'

interface SystemStatus {
  service_status: string
  database_status: string
  uptime: number
  version: string
}

interface Metrics {
  cpu_usage: number
  memory_usage: number
  disk_usage: number
  total_requests: number
  successful_requests: number
  failed_requests: number
  avg_response_time: number
}

interface LogEntry {
  id: string
  timestamp: string
  level: 'error' | 'warn' | 'info' | 'debug'
  message: string
}

const loading = ref(false)
const autoRefresh = ref(false)
const logLevel = ref('')
const logsContainer = ref()

const systemStatus = reactive<SystemStatus>({
  service_status: '正常',
  database_status: '正常',
  uptime: 0,
  version: 'v1.0.0'
})

const metrics = reactive<Metrics>({
  cpu_usage: 0,
  memory_usage: 0,
  disk_usage: 0,
  total_requests: 0,
  successful_requests: 0,
  failed_requests: 0,
  avg_response_time: 0
})

const logs = ref<LogEntry[]>([])

const filteredLogs = computed(() => {
  if (!logLevel.value) return logs.value
  return logs.value.filter(log => log.level === logLevel.value)
})

let refreshInterval: number | null = null

const formatUptime = (seconds: number) => {
  if (!seconds) return '0秒'
  
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  
  if (days > 0) {
    return `${days}天 ${hours}小时 ${minutes}分钟`
  } else if (hours > 0) {
    return `${hours}小时 ${minutes}分钟`
  } else {
    return `${minutes}分钟`
  }
}

const getProgressColor = (percentage: number) => {
  if (percentage < 50) return '#67c23a'
  if (percentage < 80) return '#e6a23c'
  return '#f56c6c'
}

const loadSystemStatus = async () => {
  try {
    const response = await api.monitoring.health()
    Object.assign(systemStatus, response.data)
  } catch (error) {
    console.error('加载系统状态失败:', error)
  }
}

const loadMetrics = async () => {
  try {
    const response = await api.monitoring.metrics()
    Object.assign(metrics, response.data)
  } catch (error) {
    console.error('加载性能指标失败:', error)
  }
}

const loadLogs = async () => {
  try {
    // 模拟日志数据，实际应该从API获取
    const mockLogs: LogEntry[] = [
      {
        id: Date.now().toString(),
        timestamp: new Date().toISOString(),
        level: 'info',
        message: '用户登录成功 - user_id: 12345'
      },
      {
        id: (Date.now() - 1000).toString(),
        timestamp: new Date(Date.now() - 1000).toISOString(),
        level: 'warn',
        message: 'API 请求频率过高 - IP: 192.168.1.100'
      },
      {
        id: (Date.now() - 2000).toString(),
        timestamp: new Date(Date.now() - 2000).toISOString(),
        level: 'error',
        message: '数据库连接超时 - connection_id: conn_001'
      }
    ]
    
    // 添加新日志到顶部，保持最新的100条
    logs.value = [...mockLogs, ...logs.value].slice(0, 100)
    
    // 自动滚动到底部
    if (logsContainer.value) {
      logsContainer.value.scrollTop = logsContainer.value.scrollHeight
    }
  } catch (error) {
    console.error('加载日志失败:', error)
  }
}

const refreshData = async () => {
  loading.value = true
  try {
    await Promise.all([
      loadSystemStatus(),
      loadMetrics(),
      loadLogs()
    ])
  } catch (error) {
    ElMessage.error('刷新数据失败')
  } finally {
    loading.value = false
  }
}

const clearLogs = () => {
  logs.value = []
  ElMessage.success('日志已清空')
}

const toggleAutoRefresh = () => {
  autoRefresh.value = !autoRefresh.value
  
  if (autoRefresh.value) {
    refreshInterval = window.setInterval(() => {
      loadMetrics()
      loadLogs()
    }, 5000) // 每5秒刷新一次
    ElMessage.success('已开启自动刷新')
  } else {
    if (refreshInterval) {
      clearInterval(refreshInterval)
      refreshInterval = null
    }
    ElMessage.success('已停止自动刷新')
  }
}

onMounted(() => {
  refreshData()
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})
</script>

<style scoped>
.monitoring {
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

.status-cards {
  margin-bottom: 20px;
}

.status-card {
  height: 100px;
}

.status-item {
  display: flex;
  align-items: center;
  height: 100%;
  gap: 16px;
}

.status-icon {
  width: 50px;
  height: 50px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  color: white;
  background: var(--el-color-primary);
}

.status-icon.healthy {
  background: var(--el-color-success);
}

.status-content {
  flex: 1;
}

.status-title {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
}

.status-value {
  font-size: 18px;
  font-weight: bold;
  color: var(--el-text-color-primary);
}

.metrics-section {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.metrics-grid {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.metric-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.metric-label {
  font-size: 14px;
  color: var(--el-text-color-secondary);
}

.api-stats {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.stat-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.stat-label {
  font-size: 14px;
  color: var(--el-text-color-secondary);
}

.stat-value {
  font-size: 16px;
  font-weight: bold;
  color: var(--el-text-color-primary);
}

.stat-value.success {
  color: var(--el-color-success);
}

.stat-value.error {
  color: var(--el-color-danger);
}

.logs-section {
  margin-bottom: 20px;
}

.log-controls {
  display: flex;
  align-items: center;
}

.logs-container {
  height: 400px;
  overflow-y: auto;
  background: #f8f9fa;
  border-radius: 4px;
  padding: 10px;
  font-family: 'Courier New', monospace;
}

.empty-logs {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--el-text-color-secondary);
}

.log-entry {
  display: flex;
  gap: 12px;
  padding: 4px 0;
  border-bottom: 1px solid #eee;
  font-size: 12px;
}

.log-entry:last-child {
  border-bottom: none;
}

.log-time {
  color: #666;
  width: 80px;
  flex-shrink: 0;
}

.log-level {
  width: 60px;
  flex-shrink: 0;
  font-weight: bold;
}

.log-entry.error .log-level {
  color: var(--el-color-danger);
}

.log-entry.warn .log-level {
  color: var(--el-color-warning);
}

.log-entry.info .log-level {
  color: var(--el-color-primary);
}

.log-entry.debug .log-level {
  color: var(--el-color-info);
}

.log-message {
  flex: 1;
  word-break: break-all;
}
</style>