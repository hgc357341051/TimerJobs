<template>
  <div class="home">
    <div class="header-actions" style="margin-bottom: 20px; text-align: right;">
      <el-switch
        v-model="autoRefreshEnabled"
        active-text="自动刷新系统状态"
        inactive-text="停止自动刷新"
        @change="toggleAutoRefresh"
        style="margin-right: 20px"
      />
      <el-button type="primary" @click="loadData" :loading="loading" icon="Refresh">
        刷新全部数据
      </el-button>
      <span style="color: #909399; font-size: 14px; margin-left: 20px">
        最后更新: {{ lastUpdateTime }}
      </span>
    </div>
    <el-row :gutter="20">
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <el-icon class="stat-icon" color="#409EFF"><Clock /></el-icon>
            <div class="stat-info">
              <div class="stat-number">{{ stats.totalJobs }}</div>
              <div class="stat-label">总任务数</div>
            </div>
          </div>
        </el-card>
      </el-col>
      
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <el-icon class="stat-icon" color="#67C23A"><SuccessFilled /></el-icon>
            <div class="stat-info">
              <div class="stat-number">{{ stats.runningJobs }}</div>
              <div class="stat-label">运行中</div>
            </div>
          </div>
        </el-card>
      </el-col>
      
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <el-icon class="stat-icon" color="#E6A23C"><WarningFilled /></el-icon>
            <div class="stat-info">
              <div class="stat-number">{{ stats.stoppedJobs }}</div>
              <div class="stat-label">已停止</div>
            </div>
          </div>
        </el-card>
      </el-col>
      
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <el-icon class="stat-icon" color="#909399"><Clock /></el-icon>
            <div class="stat-info">
              <div class="stat-number">{{ stats.waitingJobs }}</div>
              <div class="stat-label">等待</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
        <el-card title="系统监控">
          <template #header>
            <div class="card-header">
              <span style="margin-right: 20px;">系统监控</span>
              <el-button type="primary" size="small" @click="showIpDialog = true">添加IP</el-button>
            </div>
          </template>
          
          <el-descriptions title="系统状态" :column="1" border style="margin-bottom: 16px;">
            <el-descriptions-item label="系统状态">
              <el-tag :type="systemHealth.status === 'ok' ? 'success' : 'danger'">
                {{ systemHealth.status === 'ok' ? '正常' : '异常' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="运行时间">{{ systemHealth.uptime }}</el-descriptions-item>
            <el-descriptions-item label="内存使用">{{ systemHealth.memory }}</el-descriptions-item>
            <el-descriptions-item label="协程数量">{{ systemHealth.goroutines }}</el-descriptions-item>
          </el-descriptions>

          <el-descriptions title="IP控制" :column="1" border>
            <el-descriptions-item label="统计">
              <el-tag type="success">白名单 {{ ipWhitelist.length }} 个</el-tag>
              <el-tag type="danger" style="margin-left: 8px;">黑名单 {{ ipBlacklist.length }} 个</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="IP白名单">
              <el-tag
                v-for="ip in ipWhitelist"
                :key="ip"
                closable
                @close="removeIp(ip, 'whitelist')"
                style="margin-right: 8px; margin-bottom: 4px;"
              >
                {{ ip }}
              </el-tag>
              <span v-if="ipWhitelist.length === 0" style="color: #909399;">暂无白名单IP</span>
            </el-descriptions-item>
            <el-descriptions-item label="IP黑名单">
              <el-tag
                v-for="ip in ipBlacklist"
                :key="ip"
                type="danger"
                closable
                @close="removeIp(ip, 'blacklist')"
                style="margin-right: 8px; margin-bottom: 4px;"
              >
                {{ ip }}
              </el-tag>
              <span v-if="ipBlacklist.length === 0" style="color: #909399;">暂无黑名单IP</span>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
        <el-card title="系统管理">
          <template #header>
            <div class="card-header">
              <span style="margin-right: 20px;">系统管理</span>
              <el-button type="primary" size="small" @click="reloadSystemConfig" icon="Refresh">
                重载配置
              </el-button>
            </div>
          </template>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="日志开关">
              <el-switch
                v-model="logSwitchState.zapLogSwitch"
                active-text="开启"
                inactive-text="关闭"
                @change="toggleLogSwitch"
                :disabled="true"
                style="--el-switch-on-color: #13ce66; --el-switch-off-color: #ff4949"
              />
            </el-descriptions-item>
            <el-descriptions-item label="并发模式">
              <el-tag :type="systemConfig.manual_allow_concurrent ? 'success' : 'warning'">
                {{ systemConfig.manual_allow_concurrent ? '允许并发' : '禁止并发' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="默认超时">
              <el-tag>{{ systemConfig.default_timeout_seconds }}秒</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="日志摘要">
              <el-tag :type="systemConfig.log_summary_enabled ? 'success' : 'danger'">
                {{ systemConfig.log_summary_enabled ? '已启用' : '已禁用' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="响应限制">
              <el-tag>{{ (systemConfig.http_response_max_bytes / 1024 / 1024).toFixed(1) }}MB</el-tag>
            </el-descriptions-item>
          </el-descriptions>
          <div style="margin-top: 16px; text-align: center;">
            <el-button type="primary" plain @click="openConfigDialog" icon="Setting">
              查看详细配置
            </el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
        <el-card title="调度器任务">
          <template #header>
            <div class="card-header">
              <span style="margin-right: 8px;">调度器任务</span>
              <span>
                <el-tag 
                  :type="schedulerInfo.running ? 'success' : 'danger'" 
                  size="small"
                  effect="dark"
                  style="margin-right: 8px;"
                >
                  {{ schedulerInfo.running ? '运行中' : '已停止' }}
                </el-tag>
                <el-tag type="info" size="small">{{ schedulerTasks.length }} 个任务</el-tag>
              </span>
            </div>
          </template>
          <el-table :data="schedulerTasks" style="width: 100%" v-loading="loading" empty-text="暂无调度器任务">
            <el-table-column prop="name" label="任务名称" min-width="120" />
            <el-table-column prop="cron_expr" label="执行规则" min-width="100" />
            <el-table-column prop="next_run" label="下次执行" min-width="150" />
            <el-table-column prop="run_count" label="执行次数" width="80" align="center" />
            <el-table-column prop="state" label="状态" width="80" align="center">
              <template #default="scope">
                <el-tag :type="scope.row.state === 1 ? 'success' : scope.row.state === 2 ? 'danger' : 'warning'">
              {{ getStateText(scope.row.state) }}
            </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="80" align="center">
              <template #default="scope">
                <el-button 
                  link 
                  type="primary" 
                  @click="viewJobDetail(scope.row)"
                  size="small"
                >
                  详情
                </el-button>
              </template>
            </el-table-column>
          </el-table>
          <div style="margin-top: 16px; text-align: center;">
            <el-pagination
              v-model:current-page="schedulerPagination.page"
              v-model:page-size="schedulerPagination.size"
              :page-sizes="[10, 20, 50, 100]"
              :total="schedulerPagination.total"
              layout="total, sizes, prev, pager, next"
              @size-change="handleSchedulerSizeChange"
              @current-change="handleSchedulerCurrentChange"
            />
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- IP控制管理对话框 -->
    <el-dialog v-model="showIpDialog" title="添加IP控制" width="400px">
      <el-form :model="ipForm" label-width="80px">
        <el-form-item label="IP地址">
          <el-input v-model="ipForm.ip" placeholder="请输入IP地址" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="ipForm.type" style="width: 100%">
            <el-option label="白名单" value="whitelist" />
            <el-option label="黑名单" value="blacklist" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showIpDialog = false">取消</el-button>
        <el-button type="primary" @click="addIp" :loading="ipLoading">确定</el-button>
      </template>
    </el-dialog>

    <!-- 系统配置详细信息对话框 -->
    <el-dialog v-model="showConfigDialog" title="系统配置详情" width="500px">
      <el-descriptions :column="1" border>
        <el-descriptions-item label="默认允许模式">
          <el-tag :type="systemConfig.default_allow_mode ? 'success' : 'danger'">
            {{ systemConfig.default_allow_mode ? '允许' : '禁止' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="手动并发控制">
          <el-tag :type="systemConfig.manual_allow_concurrent ? 'success' : 'warning'">
            {{ systemConfig.manual_allow_concurrent ? '允许并发' : '禁止并发' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="默认超时时间">
          <el-tag>{{ systemConfig.default_timeout_seconds }} 秒</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="HTTP响应限制">
          <el-tag>{{ systemConfig.http_response_max_bytes }} 字节</el-tag>
          <span style="margin-left: 8px; color: #909399;">
            ({{ (systemConfig.http_response_max_bytes / 1024 / 1024).toFixed(2) }} MB)
          </span>
        </el-descriptions-item>
        <el-descriptions-item label="日志摘要">
          <el-tag :type="systemConfig.log_summary_enabled ? 'success' : 'danger'">
            {{ systemConfig.log_summary_enabled ? '已启用' : '已禁用' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="日志行截断">
          <el-tag>{{ systemConfig.log_line_truncate }} 字符</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="日志开关状态">
          <el-tag :type="logSwitchState.zapLogSwitch ? 'success' : 'danger'">
            {{ logSwitchState.zapLogSwitch ? '已开启' : '已关闭' }}
          </el-tag>
        </el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button type="primary" @click="closeConfigDialog">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { jobApi, systemApi } from '@/api'
import { ElMessage } from 'element-plus'

const router = useRouter()

// 添加自动刷新开关状态
const autoRefreshEnabled = ref(true)

// 添加开关控制函数
const toggleAutoRefresh = (enabled) => {
  if (enabled) {
    startAutoRefresh()
    ElMessage.success('已开启自动刷新')
  } else {
    stopAutoRefresh()
    ElMessage.info('已关闭自动刷新')
  }
}
const stats = ref({
  totalJobs: 0,
  runningJobs: 0,
  waitingJobs: 0,
  stoppedJobs: 0
})

const systemHealth = ref({
  status: 'unknown',
  uptime: '-',
  memory: '-',
  goroutines: 0
})

const schedulerInfo = ref({
  running: false,
  total_tasks: 0
})

const schedulerTasks = ref([])
const schedulerPagination = ref({
  page: 1,
  size: 10,
  total: 0
})
const loading = ref(false)
const lastUpdateTime = ref('-')

// IP控制管理相关数据
const ipWhitelist = ref([])
const ipBlacklist = ref([])
const showIpDialog = ref(false)
const ipLoading = ref(false)
const ipForm = ref({
  ip: '',
  type: 'whitelist'
})

// 系统管理相关数据
const systemConfig = ref({
  default_allow_mode: true,
  manual_allow_concurrent: true,
  default_timeout_seconds: 30,
  http_response_max_bytes: 1048576,
  log_summary_enabled: true,
  log_line_truncate: 1000
})
const logSwitchState = ref({
  zapLogSwitch: false
})
const showConfigDialog = ref(false)

const getStateText = (state) => {
  const stateMap = {
    0: '等待',
    1: '运行中',
    2: '已停止'
  }
  return stateMap[state] || '未知'
}

// 添加内存格式化函数
const formatMemory = (bytes) => {
  if (bytes === 0) return '0 B'
  
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const loadData = async () => {
  loading.value = true
  try {
    // 获取系统健康状态
    const response = await systemApi.getHealth()
    const healthData = response.data
    
    systemHealth.value = {
      status: 'ok',
      uptime: formatUptime(healthData.uptime || 0),
      memory: formatMemory(healthData.memory?.alloc || 0) + ' / ' + formatMemory(healthData.memory?.sys || 0),
      goroutines: healthData.goroutines || 0
    }

    // 计算统计信息
    const allJobs = await jobApi.getJobs()
    let allJobData = []
    if (Array.isArray(allJobs.data)) {
      allJobData = allJobs.data || []
    } else {
      allJobData = allJobs.data.list || []
    }
    const jobList = allJobData
    
    stats.value.totalJobs = jobList.length
    stats.value.runningJobs = jobList.filter(job => job.state === 1).length
    stats.value.waitingJobs = jobList.filter(job => job.state === 0).length
    stats.value.stoppedJobs = jobList.filter(job => job.state === 2).length

    // 获取调度器任务信息
    const schedulerParams = {
      page: schedulerPagination.value.page,
      size: schedulerPagination.value.size
    }
    const schedulerResponse = await jobApi.getSchedulerTasks(schedulerParams)
    if (Array.isArray(schedulerResponse.data)) {
      // 扁平式数据结构
      schedulerTasks.value = schedulerResponse.data || []
      schedulerPagination.value.total = schedulerResponse.total || 0
      schedulerInfo.value = {
        running: true, // 默认认为调度器运行中
        total_tasks: schedulerResponse.total || 0
      }
    } else if (schedulerResponse.data && schedulerResponse.data.list) {
      // 分页格式
      schedulerInfo.value = {
        running: schedulerResponse.data.running || false,
        total_tasks: schedulerResponse.data.total || 0
      }
      schedulerTasks.value = schedulerResponse.data.list || []
      schedulerPagination.value.total = schedulerResponse.data.total || 0
    } else if (schedulerResponse.data && schedulerResponse.data.tasks) {
      // 兼容旧格式
      schedulerInfo.value = {
        running: schedulerResponse.data.scheduler_running || false,
        total_tasks: schedulerResponse.data.total_tasks || 0
      }
      schedulerTasks.value = schedulerResponse.data.tasks || []
      schedulerPagination.value.total = schedulerResponse.data.total_tasks || 0
    } else {
      // 兼容最旧格式
      schedulerInfo.value = schedulerResponse.data || { running: false, total_tasks: 0 }
      schedulerTasks.value = []
      schedulerPagination.value.total = 0
    }

    // 获取IP控制信息
    try {
      const ipResponse = await systemApi.getIPControlStatus()
      ipWhitelist.value = ipResponse.data?.whitelist || []
      ipBlacklist.value = ipResponse.data?.blacklist || []
    } catch (error) {
      console.warn('获取IP控制信息失败:', error.message)
    }

    // 获取系统配置
    try {
      const configResponse = await jobApi.getJobsConfig()
      systemConfig.value = { ...systemConfig.value, ...configResponse.data }
    } catch (error) {
      console.warn('获取系统配置失败:', error.message)
    }

    // 获取日志开关状态
    try {
      const logResponse = await jobApi.getLogSwitchState()
      logSwitchState.value = { ...logResponse.data }
    } catch (error) {
      console.warn('获取日志开关状态失败:', error.message)
    }

    lastUpdateTime.value = new Date().toLocaleString()
  } catch (error) {
    ElMessage.error('获取数据失败: ' + error.message)
    systemHealth.value.status = 'error'
  } finally {
    loading.value = false
  }
}

// 添加运行时间格式化函数
const formatUptime = (seconds) => {
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = seconds % 60
  
  if (hours > 0) {
    return `${hours}小时${minutes}分钟${secs}秒`
  } else if (minutes > 0) {
    return `${minutes}分钟${secs}秒`
  } else {
    return `${secs}秒`
  }
}

// 格式化调度器任务时间
const formatSchedulerTime = (time) => {
  if (!time) return '-'
  try {
    const date = new Date(time)
    return date.toLocaleString('zh-CN')
  } catch (error) {
    return time
  }
}

// 系统管理相关方法
const reloadSystemConfig = async () => {
  try {
    await systemApi.reloadConfig()
    ElMessage.success('配置重载成功')
    await loadData() // 重新加载所有数据
  } catch (error) {
    ElMessage.error('配置重载失败: ' + error.message)
  }
}

const toggleLogSwitch = async (value) => {
  // 注意：这个接口可能需要后端支持才能修改，目前只是显示状态
  console.log('日志开关状态:', value)
}

const openConfigDialog = () => {
  showConfigDialog.value = true
}

const closeConfigDialog = () => {
  showConfigDialog.value = false
}

const refreshScheduler = async () => {
  loading.value = true
  try {
    await jobApi.calibrateJobs()
    await loadData()
    ElMessage.success('调度器已刷新')
  } catch (error) {
    ElMessage.error('刷新失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

const handleSchedulerSizeChange = (size) => {
  schedulerPagination.value.size = size
  loadData()
}

const handleSchedulerCurrentChange = (page) => {
  schedulerPagination.value.page = page
  loadData()
}

const viewJobDetail = (job) => {
  router.push(`/jobs/${job.id}`)
}

// IP控制管理相关方法
const validateIP = (ip) => {
  const ipv4Regex = /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/
  const ipv6Regex = /^(([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))$/
  return ipv4Regex.test(ip) || ipv6Regex.test(ip)
}

const addIp = async () => {
  if (!ipForm.value.ip) {
    ElMessage.warning('请输入IP地址')
    return
  }
  
  if (!validateIP(ipForm.value.ip)) {
    ElMessage.warning('请输入有效的IP地址')
    return
  }
  
  // 检查是否已存在
  const exists = ipWhitelist.value.includes(ipForm.value.ip) || ipBlacklist.value.includes(ipForm.value.ip)
  if (exists) {
    ElMessage.warning('该IP地址已存在')
    return
  }
  
  ipLoading.value = true
  try {
    if (ipForm.value.type === 'whitelist') {
      await systemApi.addToWhitelist(ipForm.value.ip)
    } else {
      await systemApi.addToBlacklist(ipForm.value.ip)
    }
    ElMessage.success('添加成功')
    showIpDialog.value = false
    ipForm.value.ip = ''
    ipForm.value.type = 'whitelist'
    await loadData()
  } catch (error) {
    ElMessage.error('添加失败: ' + error.message)
  } finally {
    ipLoading.value = false
  }
}

const removeIp = async (ip, type) => {
  try {
    if (type === 'whitelist') {
      await systemApi.removeFromWhitelist(ip)
    } else {
      await systemApi.removeFromBlacklist(ip)
    }
    ElMessage.success('删除成功')
    await loadData()
  } catch (error) {
    ElMessage.error('删除失败: ' + error.message)
  }
}

// 添加定时器引用
let refreshInterval = null

// 添加启动自动刷新的函数 - 仅刷新系统健康状态
const startAutoRefresh = () => {
  // 先清除可能存在的定时器
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
  
  // 设置5秒自动刷新，仅刷新系统健康状态
  refreshInterval = setInterval(async () => {
    try {
      const response = await systemApi.getHealth()
      const healthData = response.data
      
      systemHealth.value = {
        status: 'ok',
        uptime: formatUptime(healthData.uptime || 0),
        memory: formatMemory(healthData.memory?.alloc || 0) + ' / ' + formatMemory(healthData.memory?.sys || 0),
        goroutines: healthData.goroutines || 0
      }
      
      lastUpdateTime.value = new Date().toLocaleString()
    } catch (error) {
      console.warn('自动刷新系统状态失败:', error.message)
      systemHealth.value.status = 'error'
    }
  }, 5000)
}

// 添加停止自动刷新的函数
const stopAutoRefresh = () => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
    refreshInterval = null
  }
}

onMounted(() => {
  loadData()
  startAutoRefresh() // 启动自动刷新
})

// 在组件卸载时清理定时器
onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<style scoped>
 .home {
  padding:20px;
}

.stat-card {
  text-align: center;
}

.stat-content {
  display: flex;
  align-items: center;
  justify-content: center;
}

.stat-icon {
  font-size: 48px;
  margin-right: 20px;
}

.stat-number {
  font-size: 32px;
  font-weight: bold;
  color: #303133;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-top: 5px;
}

.card-header {
  font-weight: bold;
}
</style>