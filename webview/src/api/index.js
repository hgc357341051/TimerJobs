import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 10000
})

// 请求拦截器
api.interceptors.request.use(
  config => {
    return config
  },
  error => {
    return Promise.reject(error)
  }
)

// 响应拦截器
api.interceptors.response.use(
  response => {
    return response.data
  },
  error => {
    console.error('API Error:', error)
    return Promise.reject(error)
  }
)

// 系统API
export const systemApi = {
  // 获取系统健康状态
  getHealth() {
    return api.get('/jobs/health')
  },
  
  // 获取数据库信息
  getDatabaseInfo() {
    return api.get('/jobs/database')
  },
  
  // 获取系统状态
  getSystemStatus() {
    return api.get('/jobs/system-status')
  },
  
  // 重新加载配置
  reloadConfig() {
    return api.post('/jobs/reload-config')
  },
  
  // 获取IP控制状态
  getIPControlStatus() {
    return api.get('/jobs/ip-control/status')
  },
  
  // 添加白名单IP
  addToWhitelist(ip) {
    return api.post('/jobs/ip-control/whitelist/add', { ip })
  },
  
  // 移除白名单IP
  removeFromWhitelist(ip) {
    return api.post('/jobs/ip-control/whitelist/remove', { ip })
  },
  
  // 添加黑名单IP
  addToBlacklist(ip) {
    return api.post('/jobs/ip-control/blacklist/add', { ip })
  },
  
  // 移除黑名单IP
  removeFromBlacklist(ip) {
    return api.post('/jobs/ip-control/blacklist/remove', { ip })
  }
}

// 任务API
export const jobApi = {
  // 获取任务列表
  getJobs(params = {}) {
    return api.get('/jobs/list', { params })
  },
  
  // 获取单个任务详情
  getJob(id) {
    return api.get('/jobs/read', { params: { id } })
  },
  
  // 添加任务
  addJob(data) {
    return api.post('/jobs/add', data)
  },
  
  // 编辑任务
  editJob(data) {
    return api.post('/jobs/edit', data)
  },
  
  // 删除任务
  deleteJob(id) {
    return api.post('/jobs/del', { id })
  },
  
  // 停止任务
  stopJob(id) {
    return api.post('/jobs/stop', { id })
  },
  
  // 运行任务
  runJob(id) {
    return api.post('/jobs/run', { id })
  },
  
  // 重启任务
  restartJob(id) {
    return api.post('/jobs/restart', { id })
  },
  
  // 停止所有任务
  stopAllJobs() {
    return api.post('/jobs/stopAll')
  },
  
  // 运行所有任务
  runAllJobs() {
    return api.post('/jobs/runAll')
  },
  
  // 获取任务日志
  getJobLogs(params) {
    return api.post('/jobs/logs', params)
  },
  
  // 获取任务执行记录
  getExecByID(id) {
    return api.get('/jobs/execs', { params: { id } })
  },
  
  // 清除日志
  clearLogs() {
    return api.post('/jobs/logs/clear')
  },
  
  // 获取任务状态
  getJobState() {
    return api.get('/jobs/jobState')
  },
  
  // 获取调度器任务
  getSchedulerTasks() {
    return api.get('/jobs/scheduler')
  },
  
  // 校准任务列表
  calibrateJobs() {
    return api.post('/jobs/checkJob')
  },
  
  // 获取可用函数
  getFunctions() {
    return api.get('/jobs/functions')
  },
  
  // 获取任务配置
  getJobsConfig() {
    return api.get('/jobs/config')
  },
  
  // 获取日志开关状态
  getLogSwitchState() {
    return api.get('/jobs/switchState')
  },
  
  // 获取Zap日志
  getZapLogs(params) {
    return api.get('/jobs/zapLogs', { params })
  }
}

export default api