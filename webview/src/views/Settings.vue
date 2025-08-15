<template>
  <div class="settings">
    <el-row :gutter="20">
      <!-- 系统配置 -->
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>系统配置</span>
              <el-button @click="handleReloadConfig" :loading="loading">
                <el-icon><Refresh /></el-icon>
                重新加载
              </el-button>
            </div>
          </template>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="应用名称">{{ config.app?.name || '-' }}</el-descriptions-item>
            <el-descriptions-item label="应用版本">{{ config.app?.version || '-' }}</el-descriptions-item>
            <el-descriptions-item label="监听端口">{{ config.server?.port || '-' }}</el-descriptions-item>
            <el-descriptions-item label="运行模式">{{ config.app?.mode || '-' }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>

      <!-- 数据库信息 -->
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>数据库信息</span>
              <el-button @click="fetchDatabaseInfo" :loading="loading">
                <el-icon><Refresh /></el-icon>
              </el-button>
            </div>
          </template>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="数据库类型">{{ databaseInfo.type || '-' }}</el-descriptions-item>
            <el-descriptions-item label="主机地址">{{ databaseInfo.host || '-' }}</el-descriptions-item>
            <el-descriptions-item label="数据库名">{{ databaseInfo.database || '-' }}</el-descriptions-item>
            <el-descriptions-item label="连接状态">
              <el-tag :type="databaseInfo.status === 'open' ? 'success' : 'danger'">
                {{ databaseInfo.status === 'open' ? '已连接' : '未连接' }}
              </el-tag>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>

    <!-- 系统状态 -->
    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>系统状态</span>
              <el-button @click="fetchSystemStatus" :loading="loading">
                <el-icon><Refresh /></el-icon>
              </el-button>
            </div>
          </template>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="CPU使用率">{{ systemStatus.cpu_percent || '-' }}%</el-descriptions-item>
            <el-descriptions-item label="内存使用率">{{ systemStatus.memory_percent || '-' }}%</el-descriptions-item>
            <el-descriptions-item label="磁盘使用率">{{ systemStatus.disk_percent || '-' }}%</el-descriptions-item>
            <el-descriptions-item label="系统负载">{{ systemStatus.load_average || '-' }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>

      <!-- IP控制设置 -->
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>IP控制设置</span>
              <el-button @click="fetchIPControlStatus" :loading="loading">
                <el-icon><Refresh /></el-icon>
              </el-button>
            </div>
          </template>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="白名单">
              <el-tag 
                v-for="ip in ipControl.whitelist" 
                :key="ip" 
                closable
                @close="removeFromWhitelist(ip)"
                style="margin-right: 5px; margin-bottom: 5px;"
              >
                {{ ip }}
              </el-tag>
              <el-input
                v-if="showWhitelistInput"
                v-model="whitelistIP"
                size="small"
                style="width: 120px"
                @keyup.enter="addToWhitelist"
                @blur="showWhitelistInput = false"
                placeholder="IP地址"
              />
              <el-button v-else size="small" @click="showWhitelistInput = true">
                <el-icon><Plus /></el-icon>
                添加
              </el-button>
            </el-descriptions-item>
            <el-descriptions-item label="黑名单">
              <el-tag 
                v-for="ip in ipControl.blacklist" 
                :key="ip" 
                type="danger"
                closable
                @close="removeFromBlacklist(ip)"
                style="margin-right: 5px; margin-bottom: 5px;"
              >
                {{ ip }}
              </el-tag>
              <el-input
                v-if="showBlacklistInput"
                v-model="blacklistIP"
                size="small"
                style="width: 120px"
                @keyup.enter="addToBlacklist"
                @blur="showBlacklistInput = false"
                placeholder="IP地址"
              />
              <el-button v-else size="small" @click="showBlacklistInput = true">
                <el-icon><Plus /></el-icon>
                添加
              </el-button>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>

    <!-- 调度器任务显示区域 -->
    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>调度器任务</span>
              <div>
                <el-button 
                  @click="handleCalibrateJobs" 
                  :loading="calibrateLoading"
                  type="warning"
                >
                  <el-icon><Operation /></el-icon>
                  校准任务
                </el-button>
                <el-button @click="fetchSchedulerTasks" :loading="schedulerLoading">
                  <el-icon><Refresh /></el-icon>
                  刷新
                </el-button>
              </div>
            </div>
          </template>

          <el-row :gutter="20" style="margin-bottom: 20px;">
            <el-col :span="6">
              <div class="stat-item">
                <div class="stat-label">调度器状态</div>
                <div class="stat-value">
                  <el-tag :type="schedulerInfo.running ? 'success' : 'danger'">
                    {{ schedulerInfo.running ? '运行中' : '已停止' }}
                  </el-tag>
                </div>
              </div>
            </el-col>
            <el-col :span="6">
              <div class="stat-item">
                <div class="stat-label">任务总数</div>
                <div class="stat-value">{{ schedulerTasks.length }} 个</div>
              </div>
            </el-col>
            <el-col :span="6">
              <div class="stat-item">
                <div class="stat-label">运行中任务</div>
                <div class="stat-value">
                  <el-tag type="success">
                    {{ schedulerTasks.filter(t => t.state === 1).length }} 个
                  </el-tag>
                </div>
              </div>
            </el-col>
            <el-col :span="6">
              <div class="stat-item">
                <div class="stat-label">已停止任务</div>
                <div class="stat-value">
                  <el-tag type="info">
                    {{ schedulerTasks.filter(t => t.state === 0).length }} 个
                  </el-tag>
                </div>
              </div>
            </el-col>
          </el-row>

          <el-table :data="schedulerTasks" style="width: 100%" v-loading="schedulerLoading">
            <el-table-column prop="id" label="任务ID" width="80" />
            <el-table-column prop="name" label="任务名称" />
            <el-table-column prop="cron_expr" label="执行规则" />
            <el-table-column prop="mode" label="执行模式" width="100">
              <template #default="scope">
                <el-tag>{{ getModeText(scope.row.mode) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="state" label="状态" width="100">
              <template #default="scope">
                <el-tag :type="getStateType(scope.row.state)">
                  {{ getStateText(scope.row.state) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建时间" width="180">
              <template #default="scope">
                {{ formatTime(scope.row.created_at) }}
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100" fixed="right">
              <template #default="scope">
                <el-button 
                  link 
                  type="primary" 
                  @click="viewTaskDetail(scope.row)"
                >
                  详情
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { systemApi, jobApi } from '@/api'
import { ElMessage } from 'element-plus'

const config = ref({})
const databaseInfo = ref({})
const systemStatus = ref({})
const ipControl = ref({
  whitelist: [],
  blacklist: []
})
const schedulerTasks = ref([])
const schedulerInfo = ref({
  running: false,
  total_tasks: 0
})

const loading = ref(false)
const schedulerLoading = ref(false)
const calibrateLoading = ref(false)

const showWhitelistInput = ref(false)
const showBlacklistInput = ref(false)
const whitelistIP = ref('')
const blacklistIP = ref('')

const getModeText = (mode) => {
  const modeMap = {
    command: '命令行',
    http: 'HTTP请求',
    function: '函数调用'
  }
  return modeMap[mode] || mode
}

const getStateText = (state) => {
  const stateMap = {
    0: '已停止',
    1: '运行中',
    2: '异常'
  }
  return stateMap[state] || '未知'
}

const getStateType = (state) => {
  const typeMap = {
    0: 'info',
    1: 'success',
    2: 'danger'
  }
  return typeMap[state] || 'info'
}

const formatTime = (time) => {
  if (!time) return '-'
  return new Date(time).toLocaleString()
}

const fetchDatabaseInfo = async () => {
  try {
    const response = await systemApi.getDatabaseInfo()
    databaseInfo.value = response.data || {}
  } catch (error) {
    ElMessage.error('获取数据库信息失败: ' + error.message)
  }
}

const fetchSystemStatus = async () => {
  try {
    const response = await systemApi.getSystemStatus()
    systemStatus.value = response.data || {}
  } catch (error) {
    ElMessage.error('获取系统状态失败: ' + error.message)
  }
}

const fetchIPControlStatus = async () => {
  try {
    const response = await systemApi.getIPControlStatus()
    ipControl.value = response.data || { whitelist: [], blacklist: [] }
  } catch (error) {
    ElMessage.error('获取IP控制状态失败: ' + error.message)
  }
}

const fetchSchedulerTasks = async () => {
  schedulerLoading.value = true
  try {
    const response = await jobApi.getSchedulerTasks()
    schedulerTasks.value = response.data.tasks || []
    schedulerInfo.value = {
      running: response.data.running || false,
      total_tasks: response.data.total_tasks || 0
    }
  } catch (error) {
    ElMessage.error('获取调度器任务失败: ' + error.message)
  } finally {
    schedulerLoading.value = false
  }
}

const handleCalibrateJobs = async () => {
  calibrateLoading.value = true
  try {
    await jobApi.calibrateJobs()
    ElMessage.success('任务校准完成')
    await fetchSchedulerTasks()
  } catch (error) {
    ElMessage.error('校准任务失败: ' + error.message)
  } finally {
    calibrateLoading.value = false
  }
}

const addToWhitelist = async () => {
  if (!whitelistIP.value) return
  
  try {
    await systemApi.addToWhitelist(whitelistIP.value)
    ElMessage.success('添加成功')
    whitelistIP.value = ''
    showWhitelistInput.value = false
    await fetchIPControlStatus()
  } catch (error) {
    ElMessage.error('添加失败: ' + error.message)
  }
}

const removeFromWhitelist = async (ip) => {
  try {
    await systemApi.removeFromWhitelist(ip)
    ElMessage.success('移除成功')
    await fetchIPControlStatus()
  } catch (error) {
    ElMessage.error('移除失败: ' + error.message)
  }
}

const addToBlacklist = async () => {
  if (!blacklistIP.value) return
  
  try {
    await systemApi.addToBlacklist(blacklistIP.value)
    ElMessage.success('添加成功')
    blacklistIP.value = ''
    showBlacklistInput.value = false
    await fetchIPControlStatus()
  } catch (error) {
    ElMessage.error('添加失败: ' + error.message)
  }
}

const removeFromBlacklist = async (ip) => {
  try {
    await systemApi.removeFromBlacklist(ip)
    ElMessage.success('移除成功')
    await fetchIPControlStatus()
  } catch (error) {
    ElMessage.error('移除失败: ' + error.message)
  }
}

const handleReloadConfig = async () => {
  loading.value = true
  try {
    await systemApi.reloadConfig()
    ElMessage.success('配置重新加载成功')
  } catch (error) {
    ElMessage.error('重新加载配置失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

const viewTaskDetail = (task) => {
  window.location.href = `/#/jobs/${task.id}`
}

const loadConfig = async () => {
  loading.value = true
  try {
    await Promise.all([
      fetchDatabaseInfo(),
      fetchSystemStatus(),
      fetchIPControlStatus(),
      fetchSchedulerTasks()
    ])
  } catch (error) {
    ElMessage.error('加载配置失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.settings {
  padding: 0;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.stat-item {
  text-align: center;
  padding: 15px;
  background-color: #f5f7fa;
  border-radius: 4px;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 18px;
  font-weight: bold;
  color: #303133;
}
</style>