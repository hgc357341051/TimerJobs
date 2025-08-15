<template>
  <div class="job-detail">
    <el-row :gutter="20">
      <el-col :span="16">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>任务详情</span>
              <div class="header-actions">
                <el-button @click="$router.push('/jobs')">
                  <el-icon><ArrowLeft /></el-icon>
                  返回列表
                </el-button>
              </div>
            </div>
          </template>

          <el-descriptions :column="2" border>
            <el-descriptions-item label="任务ID">{{ job.id }}</el-descriptions-item>
            <el-descriptions-item label="任务名称">{{ job.name }}</el-descriptions-item>
            <el-descriptions-item label="任务描述" :span="2">{{ job.desc || '-' }}</el-descriptions-item>
            <el-descriptions-item label="执行规则">{{ job.cron_expr }}</el-descriptions-item>
            <el-descriptions-item label="执行模式">
              <el-tag>{{ getModeText(job.mode) }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="getStateType(job.state)">
                {{ getStateText(job.state) }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="创建时间">{{ formatTime(job.created_at) }}</el-descriptions-item>
            <el-descriptions-item label="更新时间">{{ formatTime(job.updated_at) }}</el-descriptions-item>
          </el-descriptions>

          <div style="margin-top: 20px;">
            <h4>执行内容:</h4>
            <el-input
              v-model="job.command"
              type="textarea"
              :rows="4"
              readonly
            />
          </div>

          <div style="margin-top: 20px;">
            <el-button type="primary" @click="runJob" :loading="loading">
              <el-icon><VideoPlay /></el-icon>
              立即执行
            </el-button>
            <el-button 
              :type="job.state === 1 ? 'warning' : 'success'" 
              @click="toggleJobState"
              :loading="loading"
            >
              <el-icon>
                <VideoPause v-if="job.state === 1" />
                <VideoPlay v-else />
              </el-icon>
              {{ job.state === 1 ? '停止任务' : '启动任务' }}
            </el-button>
            <el-button type="danger" @click="deleteJob" :loading="loading">
              <el-icon><Delete /></el-icon>
              删除任务
            </el-button>
          </div>
        </el-card>
      </el-col>

      <el-col :span="8">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>执行记录</span>
              <el-button link @click="loadExecutions">
                <el-icon><Refresh /></el-icon>
              </el-button>
            </div>
          </template>
          
          <el-timeline v-loading="execLoading">
            <el-timeline-item
              v-for="exec in executions"
              :key="exec.id"
              :timestamp="formatTime(exec.created_at)"
              :type="exec.status === 'success' ? 'success' : 'danger'"
            >
              <div>
                <div>状态: 
                  <el-tag :type="exec.status === 'success' ? 'success' : 'danger'" size="small">
                    {{ exec.status }}
                  </el-tag>
                </div>
                <div v-if="exec.duration">耗时: {{ exec.duration }}ms</div>
                <div v-if="exec.error_msg" class="error-msg">
                  错误: {{ exec.error_msg }}
                </div>
              </div>
            </el-timeline-item>
          </el-timeline>
          
          <div v-if="executions.length === 0" class="empty-state">
            <el-empty description="暂无执行记录" />
          </div>
        </el-card>

        <el-card style="margin-top: 20px;">
          <template #header>
            <div class="card-header">
              <span>任务日志</span>
              <el-button link @click="loadLogs">
                <el-icon><Refresh /></el-icon>
              </el-button>
            </div>
          </template>
          
          <div class="logs-container" v-loading="logsLoading">
            <div 
              v-for="log in logs" 
              :key="log.id"
              class="log-item"
              :class="`log-${log.level}`"
            >
              <div class="log-time">{{ formatTime(log.time) }}</div>
              <div class="log-message">{{ log.message }}</div>
            </div>
          </div>
          
          <div v-if="logs.length === 0" class="empty-state">
            <el-empty description="暂无日志" />
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { jobApi } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'

const route = useRoute()
const router = useRouter()

const job = ref({
  id: null,
  name: '',
  desc: '',
  cron_expr: '',
  mode: 'command',
  command: '',
  state: 0,
  created_at: '',
  updated_at: ''
})

const executions = ref([])
const logs = ref([])
const loading = ref(false)
const execLoading = ref(false)
const logsLoading = ref(false)

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

const loadJobDetail = async () => {
  loading.value = true
  try {
    const response = await jobApi.getJob(route.params.id)
    job.value = response.data || {}
  } catch (error) {
    ElMessage.error('加载任务详情失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

const loadExecutions = async () => {
  execLoading.value = true
  try {
    const response = await jobApi.getExecByID(route.params.id)
    executions.value = response.data || []
  } catch (error) {
    ElMessage.error('加载执行记录失败: ' + error.message)
  } finally {
    execLoading.value = false
  }
}

const loadLogs = async () => {
  logsLoading.value = true
  try {
    const response = await jobApi.getJobLogs({
      id: parseInt(route.params.id),
      limit: 20
    })
    logs.value = response.data || []
  } catch (error) {
    ElMessage.error('加载日志失败: ' + error.message)
  } finally {
    logsLoading.value = false
  }
}

const runJob = async () => {
  loading.value = true
  try {
    await jobApi.runJob(job.value.id)
    ElMessage.success('任务已开始执行')
    await loadExecutions()
  } catch (error) {
    ElMessage.error('执行任务失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

const toggleJobState = async () => {
  loading.value = true
  try {
    if (job.value.state === 1) {
      await jobApi.stopJob(job.value.id)
      ElMessage.success('任务已停止')
    } else {
      await jobApi.runJob(job.value.id)
      ElMessage.success('任务已启动')
    }
    await loadJobDetail()
  } catch (error) {
    ElMessage.error('操作失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

const deleteJob = async () => {
  try {
    await ElMessageBox.confirm(
      '确定要删除此任务吗？此操作不可恢复！',
      '警告',
      { type: 'error' }
    )
    
    loading.value = true
    await jobApi.deleteJob(job.value.id)
    ElMessage.success('任务已删除')
    router.push('/jobs')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除任务失败: ' + error.message)
    }
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadJobDetail()
  loadExecutions()
  loadLogs()
})
</script>

<style scoped>
.job-detail {
  padding: 0;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-actions {
  display: flex;
  gap: 10px;
}

.error-msg {
  color: #f56c6c;
  font-size: 12px;
  margin-top: 5px;
}

.logs-container {
  max-height: 300px;
  overflow-y: auto;
}

.log-item {
  padding: 8px 0;
  border-bottom: 1px solid #eee;
  font-size: 12px;
}

.log-item:last-child {
  border-bottom: none;
}

.log-time {
  color: #909399;
  margin-bottom: 4px;
}

.log-message {
  color: #303133;
  word-break: break-all;
}

.log-info .log-message {
  color: #909399;
}

.log-warn .log-message {
  color: #e6a23c;
}

.log-error .log-message {
  color: #f56c6c;
}

.empty-state {
  text-align: center;
  padding: 40px 0;
}
</style>