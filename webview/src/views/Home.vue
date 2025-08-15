<template>
  <div class="home">
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
            <el-icon class="stat-icon" color="#F56C6C"><CircleCloseFilled /></el-icon>
            <div class="stat-info">
              <div class="stat-number">{{ stats.errorJobs }}</div>
              <div class="stat-label">异常</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="12">
        <el-card title="系统状态">
          <template #header>
            <div class="card-header">
              <span>系统状态</span>
            </div>
          </template>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="系统状态">
              <el-tag :type="systemHealth.status === 'ok' ? 'success' : 'danger'">
                {{ systemHealth.status === 'ok' ? '正常' : '异常' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="运行时间">{{ systemHealth.uptime }}</el-descriptions-item>
            <el-descriptions-item label="内存使用">{{ systemHealth.memory }}</el-descriptions-item>
            <el-descriptions-item label="协程数量">{{ systemHealth.goroutines }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
      
      <el-col :span="12">
        <el-card title="调度器状态">
          <template #header>
            <div class="card-header">
              <span>调度器状态</span>
            </div>
          </template>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="调度器状态">
              <el-tag :type="schedulerInfo.running ? 'success' : 'danger'">
                {{ schedulerInfo.running ? '运行中' : '已停止' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="任务总数">{{ schedulerInfo.total_tasks }} 个</el-descriptions-item>
            <el-descriptions-item label="最后更新">{{ lastUpdateTime }}</el-descriptions-item>
          </el-descriptions>
          <div style="margin-top: 15px;">
            <el-button type="primary" @click="refreshScheduler" :loading="loading">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="24">
        <el-card title="最近任务">
          <template #header>
            <div class="card-header">
              <span>最近任务</span>
            </div>
          </template>
          <el-table :data="recentJobs" style="width: 100%" v-loading="loading">
            <el-table-column prop="name" label="任务名称" />
            <el-table-column prop="cron_expr" label="执行规则" />
            <el-table-column prop="state" label="状态" width="100">
              <template #default="scope">
                <el-tag :type="scope.row.state === 1 ? 'success' : scope.row.state === 0 ? 'warning' : 'danger'">
                  {{ getStateText(scope.row.state) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100">
              <template #default="scope">
                <el-button 
                  link 
                  type="primary" 
                  @click="viewJobDetail(scope.row)"
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
import { useRouter } from 'vue-router'
import { jobApi, systemApi } from '@/api'
import { ElMessage } from 'element-plus'

const router = useRouter()

const stats = ref({
  totalJobs: 0,
  runningJobs: 0,
  stoppedJobs: 0,
  errorJobs: 0
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

const recentJobs = ref([])
const loading = ref(false)
const lastUpdateTime = ref('-')

const getStateText = (state) => {
  const stateMap = {
    0: '已停止',
    1: '运行中',
    2: '异常'
  }
  return stateMap[state] || '未知'
}

const loadData = async () => {
  loading.value = true
  try {
    // 获取系统健康状态
    const health = await systemApi.getHealth()
    systemHealth.value = health.data

    // 获取任务列表
    const jobs = await jobApi.getJobs({ page: 1, size: 5 })
    recentJobs.value = jobs.data.list || []

    // 计算统计信息
    const allJobs = await jobApi.getJobs()
    const jobList = allJobs.data.list || []
    
    stats.value.totalJobs = jobList.length
    stats.value.runningJobs = jobList.filter(job => job.state === 1).length
    stats.value.stoppedJobs = jobList.filter(job => job.state === 0).length
    stats.value.errorJobs = jobList.filter(job => job.state === 2).length

    // 获取调度器信息
    const scheduler = await jobApi.getSchedulerTasks()
    schedulerInfo.value = scheduler.data || { running: false, total_tasks: 0 }

    lastUpdateTime.value = new Date().toLocaleString()
  } catch (error) {
    ElMessage.error('加载数据失败: ' + error.message)
  } finally {
    loading.value = false
  }
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

const viewJobDetail = (job) => {
  router.push(`/jobs/${job.id}`)
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.home {
  padding: 0;
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