<template>
  <div class="logs-page">
    <!-- 页面标题 -->
    <el-row :gutter="20" class="page-header">
      <el-col :span="12">
        <h2>系统日志</h2>
      </el-col>
      <el-col :span="12" class="text-right">
        <el-button type="danger" @click="clearAllLogs" :loading="loading">
          <el-icon><Delete /></el-icon>
          清除全部日志
        </el-button>
        <el-button @click="refreshLogs" :loading="loading">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </el-col>
    </el-row>

    <!-- 搜索和筛选 -->
    <el-card class="search-card">
      <el-row :gutter="20">
        <el-col :span="6">
          <el-select v-model="searchParams.level" placeholder="日志级别" clearable>
            <el-option label="全部" value="" />
            <el-option label="信息" value="info" />
            <el-option label="警告" value="warning" />
            <el-option label="错误" value="error" />
            <el-option label="调试" value="debug" />
          </el-select>
        </el-col>
   
        <el-col :span="6">
          <el-date-picker
          v-model="searchParams.date"
          type="date"
          placeholder="选择日期"
          clearable
          value-format="YYYY-MM-DD"
          @change="loadLogs"
        />
        </el-col>
        <el-col :span="6">
          <el-button type="primary" @click="loadLogs">搜索</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-col>
      </el-row>
    </el-card>



    <!-- 日志列表 -->
    <el-card>
      <el-table
        :data="logList"
        v-loading="loading"
        style="width: 100%"
        border
        height="600"
      >
        <el-table-column prop="time" label="时间" width="180" />
        <el-table-column prop="level" label="级别" width="100">
          <template #default="scope">
            <el-tag :type="getLevelType(scope.row.level)">
              {{ scope.row.level.toUpperCase() }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="caller" label="来源" width="160" />
        <el-table-column prop="msg" label="消息内容" show-overflow-tooltip />
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="scope">
            <el-button type="primary" size="small" @click="viewLogDetail(scope.row)">
              详情
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.size"
          :page-sizes="[50, 100, 200, 500]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 日志详情对话框 -->
    <el-dialog
      v-model="showDetailDialog"
      title="日志详情"
      width="600px"
    >
      <el-descriptions :column="1" border v-if="selectedLog">
        <el-descriptions-item label="时间">{{ selectedLog.time }}</el-descriptions-item>
        <el-descriptions-item label="级别">
          <el-tag :type="getLevelType(selectedLog.level)">
            {{ selectedLog.level.toUpperCase() }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="来源">{{ selectedLog.caller || '-' }}</el-descriptions-item>
        <el-descriptions-item label="关联任务">{{ selectedLog.task_name || '-' }}</el-descriptions-item>
        <el-descriptions-item label="消息内容">
          <pre>{{ selectedLog.msg }}</pre>
        </el-descriptions-item>
        <el-descriptions-item label="完整数据">
          <pre>{{ JSON.stringify(selectedLog.data || {}, null, 2) }}</pre>
        </el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showDetailDialog = false">关闭</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { jobApi } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'

// 数据
const loading = ref(false)
const logList = ref([])
const stats = ref({})
const searchParams = ref({
  level: '',
  keyword: '',
  date: ''
})
const pagination = ref({
  page: 1,
  size: 50,
  total: 0
})

// 对话框控制
const showDetailDialog = ref(false)
const selectedLog = ref(null)

// 辅助方法
const getLocalDateString = (date) => {
  // 由于使用了value-format="YYYY-MM-DD"，date已经是正确格式的字符串
  return date || ''
}

// 方法
const loadLogs = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.value.page,
      size: pagination.value.size,
      ...searchParams.value
    }
    const res = await jobApi.getZapLogs(params)
    logList.value = res.data || []
    pagination.value.total = res.total || 0
    
    // 计算统计信息
    const levelCounts = { error: 0, warning: 0, info: 0, debug: 0 }
    logList.value.forEach(log => {
      if (levelCounts.hasOwnProperty(log.level)) {
        levelCounts[log.level]++
      }
    })
    stats.value = {
      total: pagination.value.total,
      error: levelCounts.error,
      warning: levelCounts.warning,
      info: levelCounts.info,
      debug: levelCounts.debug
    }
  } catch (error) {
    ElMessage.error('获取日志失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

const loadStats = async () => {
  // 统计信息将在loadLogs中计算，这里不再单独调用
  if (logList.value.length === 0) {
    loadLogs()
  }
}

const resetSearch = () => {
  searchParams.value = { level: '', keyword: '', date: '' }
  pagination.value.page = 1
  loadLogs()
}

const handleSizeChange = (size) => {
  pagination.value.size = size
  loadLogs()
}

const handleCurrentChange = (page) => {
  pagination.value.page = page
  loadLogs()
}

const refreshLogs = () => {
  loadLogs()
  loadStats()
}

const getLevelType = (level) => {
  const typeMap = {
    error: 'danger',
    warn: 'warning',
    info: 'primary',
    debug: 'success',
    trace: 'info',
    panic: 'danger'
  }
  return typeMap[level.toLowerCase()] || 'info'
}

const viewLogDetail = (log) => {
  selectedLog.value = log
  showDetailDialog.value = true
}

const clearAllLogs = async () => {
  try {
    await ElMessageBox.confirm(
      '确定要清除所有日志吗？此操作不可恢复！',
      '确认清除',
      { type: 'warning' }
    )
    const date = searchParams.value.date ? getLocalDateString(searchParams.value.date) : ''
    await jobApi.clearLogs({ type: 'zap', date })
    ElMessage.success('日志已清除')
    loadLogs()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('清除失败: ' + error.message)
    }
  }
}

// 初始化
onMounted(() => {
  loadLogs()
})
</script>

<style scoped>
.logs-page {
  padding: 20px;
}

.page-header {
  margin-bottom: 20px;
  align-items: center;
}

.text-right {
  text-align: right;
}

.search-card {
  margin-bottom: 20px;
}

.stats-row {
  margin-bottom: 20px;
}

.stat-card {
  text-align: center;
}

.stat-content {
  padding: 10px;
}

.stat-number {
  font-size: 24px;
  font-weight: bold;
  color: #409eff;
}

.stat-number.error {
  color: #f56c6c;
}

.stat-number.warning {
  color: #e6a23c;
}

.stat-number.info {
  color: #909399;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-top: 5px;
}

.pagination {
  margin-top: 20px;
  text-align: right;
}

pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  background: #f5f5f5;
  padding: 10px;
  border-radius: 4px;
  max-height: 300px;
  overflow-y: auto;
}
</style>

<style>
/* 日志级别标签样式增强 */
.el-tag.el-tag--danger {
  background-color: #fef0f0;
  border-color: #fde2e2;
  color: #f56c6c;
  font-weight: bold;
}

.el-tag.el-tag--warning {
  background-color: #fdf6ec;
  border-color: #faecd8;
  color: #e6a23c;
  font-weight: bold;
}

.el-tag.el-tag--primary {
  background-color: #ecf5ff;
  border-color: #d9ecff;
  color: #409eff;
  font-weight: bold;
}

.el-tag.el-tag--success {
  background-color: #f0f9ff;
  border-color: #e1f3d8;
  color: #67c23a;
  font-weight: bold;
}

.el-tag.el-tag--info {
  background-color: #f4f4f5;
  border-color: #e9e9eb;
  color: #909399;
  font-weight: bold;
}
</style>