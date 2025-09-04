import axios from 'axios'
import type { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios'
import { signRequest } from '../utils/signature'

// API基础配置
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:50052'

// 创建axios实例
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器 - 添加签名
apiClient.interceptors.request.use(
  (config: AxiosRequestConfig) => {
    // 为请求添加签名
    const signedConfig = signRequest(config)
    return signedConfig
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
apiClient.interceptors.response.use(
  (response: AxiosResponse) => {
    return response.data
  },
  (error) => {
    console.error('API请求错误:', error)
    return Promise.reject(error)
  }
)

export default apiClient

// API接口定义
export interface User {
  id: string
  username: string
  email: string
  status: string
  created_at: string
  updated_at: string
}

export interface AccessKey {
  access_key_id: string
  secret_access_key: string
  status: string
  user_id: string
  created_at: string
  expires_at: string
}

export interface Policy {
  id: string
  name: string
  description: string
  policy_document: string
  created_at: string
  updated_at: string
}

export interface Application {
  id: string
  name: string
  description: string
  status: string
  created_at: string
  updated_at: string
}

export interface DeveloperVerification {
  id: string
  user_id: string
  company_name: string
  contact_email: string
  status: string
  submitted_at: string
  reviewed_at?: string
}

// API方法
export const api = {
  // 用户管理
  users: {
    create: (data: Partial<User>) => apiClient.post('/v1/users', data),
    get: (id: string) => apiClient.get(`/v1/users/${id}`),
    list: (params?: any) => apiClient.get('/v1/users', { params }),
    update: (id: string, data: Partial<User>) => apiClient.put(`/v1/users/${id}`, data),
    delete: (id: string) => apiClient.delete(`/v1/users/${id}`)
  },

  // 访问密钥管理
  accessKeys: {
    create: (data: any) => apiClient.post('/v1/access-keys', data),
    list: (params?: any) => apiClient.get('/v1/access-keys', { params }),
    updateStatus: (id: string, status: string) => 
      apiClient.put(`/v1/access-keys/${id}/status`, { status })
  },

  // 策略管理
  policies: {
    create: (data: Partial<Policy>) => apiClient.post('/v1/policies', data),
    get: (id: string) => apiClient.get(`/v1/policies/${id}`),
    list: (params?: any) => apiClient.get('/v1/policies', { params }),
    update: (id: string, data: Partial<Policy>) => apiClient.put(`/v1/policies/${id}`, data),
    delete: (id: string) => apiClient.delete(`/v1/policies/${id}`)
  },

  // 权限检查
  permissions: {
    check: (data: any) => apiClient.post('/v1/permissions/check', data),
    validate: (data: any) => apiClient.post('/v1/permissions/validate', data)
  },

  // 应用管理
  applications: {
    create: (data: Partial<Application>) => apiClient.post('/v1/applications', data),
    get: (id: string) => apiClient.get(`/v1/applications/${id}`),
    list: (params?: any) => apiClient.get('/v1/applications', { params }),
    update: (id: string, data: Partial<Application>) => apiClient.put(`/v1/applications/${id}`, data),
    delete: (id: string) => apiClient.delete(`/v1/applications/${id}`)
  },

  // 开发者认证
  developerVerification: {
    submit: (data: any) => apiClient.post('/v1/developer-verification', data),
    get: (id: string) => apiClient.get(`/v1/developer-verification/${id}`),
    list: (params?: any) => apiClient.get('/v1/developer-verification', { params }),
    review: (id: string, data: any) => apiClient.put(`/v1/developer-verification/${id}/review`, data)
  },

  // 系统监控
  monitoring: {
    health: () => apiClient.get('/v1/health'),
    metrics: () => apiClient.get('/v1/metrics')
  },

  // 配置管理
  config: {
    get: () => apiClient.get('/v1/config'),
    update: (data: any) => apiClient.put('/v1/config', data)
  },

  // 仪表板
  dashboard: {
    stats: () => apiClient.get('/v1/dashboard/stats'),
    status: () => apiClient.get('/v1/dashboard/status'),
    activities: () => apiClient.get('/v1/dashboard/activities')
  }
}