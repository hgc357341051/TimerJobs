<template>
  <div class="jobs-page">
    <!-- 页面标题和操作栏 -->
    <el-row :gutter="20" class="page-header">
      <el-col :span="12">
        <h2>任务管理</h2>
      </el-col>
      <el-col :span="12" class="text-right">
        <el-button type="primary" @click="handleAddClick">
          <el-icon><Plus /></el-icon>
          新增任务
        </el-button>
        <el-button type="success" @click="runAllJobs" :loading="loading">
          <el-icon><VideoPlay /></el-icon>
          启动全部
        </el-button>
        <el-button type="warning" @click="stopAllJobs" :loading="loading">
          <el-icon><VideoPause /></el-icon>
          停止全部
        </el-button>
        <el-button @click="calibrateJobs" :loading="loading">
          <el-icon><Refresh /></el-icon>
          校准任务
        </el-button>
      </el-col>
    </el-row>

    <!-- 搜索和筛选 -->
    <el-card class="search-card">
      <el-row :gutter="20">
        <el-col :span="6">
          <el-input
            v-model="searchParams.name"
            placeholder="任务名称"
            clearable
            @clear="loadJobs"
          />
        </el-col>
        <el-col :span="4">
          <el-select
            v-model="searchParams.state"
            placeholder="任务状态"
            clearable
            @clear="loadJobs"
          >
            <el-option label="全部" value="-1" />
            <el-option label="等待" :value="0" />
            <el-option label="运行中" :value="1" />
            <el-option label="已停止" :value="2" />
          </el-select>
        </el-col>
        <el-col :span="4">
          <el-select
            v-model="searchParams.mode"
            placeholder="执行模式"
            clearable
            @clear="loadJobs"
          >
            <el-option label="全部" value="" />
            <el-option label="HTTP" value="http" />
            <el-option label="命令" value="command" />
            <el-option label="函数" value="func" />
          </el-select>
        </el-col>
        <el-col :span="6">
          <el-button type="primary" @click="loadJobs">搜索</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-col>
      </el-row>
    </el-card>

    <!-- 任务列表 -->
    <el-card>
      <el-table
        :data="jobList"
        v-loading="loading"
        style="width: 100%"
        border
      >
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="name" label="任务名称" min-width="150" show-overflow-tooltip />
        <el-table-column prop="desc" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column prop="cron_expr" label="Cron表达式" width="120" />
        <el-table-column prop="mode" label="执行模式" width="100" />
        <el-table-column prop="state" label="状态" width="100">
          <template #default="scope">
            <el-tag :type="getStateType(scope.row.state)">
              {{ getStateText(scope.row.state) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="run_count" label="执行次数" width="100" />
        <el-table-column prop="max_run_count" label="最大次数" width="100">
          <template #default="scope">
            {{ scope.row.max_run_count || '∞' }}
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="160">
          <template #default="scope">
            {{ formatDate(scope.row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="scope">
            <el-button
              type="primary"
              size="small"
              @click="viewJobDetail(scope.row)"
            >
              详情
            </el-button>
            <el-button @click="toggleJobState(scope.row.id,1)">启动</el-button>
            <el-dropdown @command="handleCommand($event, scope.row)">
              <el-button size="small">
                更多<el-icon><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-item command="edit">编辑</el-dropdown-item>
                <el-dropdown-item command="logs">日志</el-dropdown-item>
                <el-dropdown-item command="restart">重启</el-dropdown-item>
                 <el-dropdown-item command="stop">停止</el-dropdown-item>
                <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.size"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 新增/编辑任务对话框 -->
    <el-dialog
      v-model="showAddDialog"
      :title="isEdit ? '编辑任务' : '新增任务'"
      width="800px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="jobFormRef"
        :model="jobForm"
        :rules="jobRules"
        label-width="100px"
        label-position="left"
      >
        <el-form-item label="任务名称" prop="name">
          <el-input v-model="jobForm.name" placeholder="请输入任务名称" />
        </el-form-item>
        <el-form-item label="任务描述">
          <el-input
            v-model="jobForm.desc"
            type="textarea"
            :rows="2"
            placeholder="请输入任务描述"
          />
        </el-form-item>
        <el-form-item label="Cron表达式" prop="cron_expr">
          <el-input v-model="jobForm.cron_expr" placeholder="请输入Cron表达式">
            <template #append>
              <el-button @click="showCronHelp = true" icon="Clock" />
            </template>
          </el-input>
        </el-form-item>
        <el-form-item label="执行模式" prop="mode">
          <el-select v-model="jobForm.mode" placeholder="请选择执行模式" @change="handleModeChange">
            <el-option label="HTTP请求" value="http" />
            <el-option label="shell命令" value="command" />
            <el-option label="内置函数" value="func" />
          </el-select>
        </el-form-item>

        <!-- HTTP任务配置 -->
        <template v-if="jobForm.mode === 'http'">
          <el-form-item label="URL地址" prop="command">
            <el-input v-model="commandConfig.http.url" placeholder="请输入URL地址" />
          </el-form-item>
          <el-form-item label="请求方式">
            <el-select v-model="commandConfig.http.mode">
              <el-option label="GET" value="GET" />
              <el-option label="POST" value="POST" />
              <el-option label="PUT" value="PUT" />
              <el-option label="DELETE" value="DELETE" />
            </el-select>
          </el-form-item>
          <el-form-item label="请求头">
            <el-input
              v-model="commandConfig.http.headers"
              type="textarea"
              :rows="3"
              placeholder="格式：key1:value1||key2:value2"
            />
          </el-form-item>
          <el-form-item label="POST数据">
            <el-input
              v-model="commandConfig.http.data"
              type="textarea"
              :rows="3"
              placeholder="请输入POST数据"
            />
          </el-form-item>
          <el-form-item label="Cookie">
            <el-input v-model="commandConfig.http.cookies" placeholder="请输入Cookie字符串" />
          </el-form-item>
          <el-form-item label="代理地址">
            <el-input v-model="commandConfig.http.proxy" placeholder="例如: http://proxy.example.com:8080" />
          </el-form-item>
          <el-form-item label="执行次数">
            <el-input-number v-model="commandConfig.http.times" :min="0" placeholder="0为无限制" />
          </el-form-item>
          <el-form-item label="重试间隔">
            <el-input-number v-model="commandConfig.http.interval" :min="0" placeholder="重试间隔秒数" />
          </el-form-item>
          <el-form-item label="成功判断">
            <el-input v-model="commandConfig.http.result" placeholder="自定义成功判断字符串" />
          </el-form-item>
          <el-form-item label="超时时间">
            <el-input-number v-model="commandConfig.http.timeout" :min="1" :max="300" placeholder="超时时间（秒）" />
          </el-form-item>
        </template>

        <!-- 命令任务配置 -->
        <template v-if="jobForm.mode === 'command'">
          <el-form-item label="执行命令" prop="command">
            <el-input
              v-model="commandConfig.command.command"
              type="textarea"
              :rows="3"
              placeholder="请输入要执行的命令"
            />
          </el-form-item>
          <el-form-item label="工作目录">
            <el-input v-model="commandConfig.command.workdir" placeholder="请输入工作目录路径" />
          </el-form-item>
          <el-form-item label="环境变量">
            <el-input
              v-model="commandConfig.command.env"
              type="textarea"
              :rows="3"
              placeholder="格式：key1=value1||key2=value2"
            />
          </el-form-item>
          <el-form-item label="执行次数">
            <el-input-number v-model="commandConfig.command.times" :min="0" placeholder="0为无限制" />
          </el-form-item>
          <el-form-item label="重试间隔">
            <el-input-number v-model="commandConfig.command.interval" :min="0" placeholder="重试间隔秒数" />
          </el-form-item>
          <el-form-item label="超时时间">
            <el-input-number v-model="commandConfig.command.timeout" :min="1" :max="300" placeholder="超时时间（秒）" />
          </el-form-item>
        </template>

        <!-- 函数任务配置 -->
        <template v-if="jobForm.mode === 'func'">
          <el-form-item label="函数名称" required>
            <el-select v-model="commandConfig.func.name" placeholder="请选择函数">
              <el-option
                v-for="func in functionList"
                :key="func.name"
                :label="func.name"
                :value="func.name"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="函数参数">
            <el-input
              v-model="commandConfig.func.arg"
              type="textarea"
              :rows="3"
              placeholder="请输入函数参数，用逗号分隔"
            />
          </el-form-item>
          <el-form-item label="执行次数">
            <el-input-number v-model="commandConfig.func.times" :min="0" placeholder="0为无限制" />
          </el-form-item>
          <el-form-item label="重试间隔">
            <el-input-number v-model="commandConfig.func.interval" :min="0" placeholder="重试间隔秒数" />
          </el-form-item>
          <el-form-item label="函数说明" v-if="selectedFunction">
            <el-alert :title="selectedFunction.description" type="info" :closable="false" />
          </el-form-item>
        </template>

        <el-form-item label="任务状态" prop="state">
          <el-radio-group v-model="jobForm.state">
            <el-radio :label="0">等待</el-radio>
            <el-radio :label="1">运行中</el-radio>
            <el-radio :label="2">已停止</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="执行模式" prop="allow_mode">
          <el-select v-model="jobForm.allow_mode" placeholder="请选择执行模式">
            <el-option label="并行执行（默认）" :value="0" />
            <el-option label="串行执行（跳过）" :value="1" />
            <el-option label="串行执行（排队）" :value="2" />
          </el-select>
        </el-form-item>
        <el-form-item label="最大执行次数" prop="max_run_count">
          <el-input-number
            v-model="jobForm.max_run_count"
            :min="0"
            :precision="0"
            placeholder="0为无限次"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showAddDialog = false">取消</el-button>
          <el-button type="primary" @click="submitJob" :loading="submitLoading">
            {{ isEdit ? '更新' : '创建' }}
          </el-button>
        </span>
      </template>
    </el-dialog>

    <!-- Cron表达式帮助对话框 -->
    <el-dialog v-model="showCronHelp" title="Cron表达式帮助" width="500px">
      <el-descriptions :column="1" border>
        <el-descriptions-item label="每秒执行">* * * * * *</el-descriptions-item>
        <el-descriptions-item label="每分钟执行">0 * * * * *</el-descriptions-item>
        <el-descriptions-item label="每小时执行">0 0 * * * *</el-descriptions-item>
        <el-descriptions-item label="每天0点执行">0 0 0 * * *</el-descriptions-item>
        <el-descriptions-item label="每天2点执行">0 0 2 * * *</el-descriptions-item>
        <el-descriptions-item label="每天9点30分执行">0 30 9 * * *</el-descriptions-item>
        <el-descriptions-item label="每周一0点执行">0 0 0 * * 1</el-descriptions-item>
        <el-descriptions-item label="每月1号0点执行">0 0 0 1 * *</el-descriptions-item>
      </el-descriptions>
    </el-dialog>

    <!-- 任务详情对话框 -->
    <el-dialog
      v-model="showDetailDialog"
      title="任务详情"
      width="600px"
    >
      <el-descriptions :column="1" border v-if="selectedJob">
        <el-descriptions-item label="任务ID">{{ selectedJob.id }}</el-descriptions-item>
        <el-descriptions-item label="任务名称">{{ selectedJob.name }}</el-descriptions-item>
        <el-descriptions-item label="任务描述">{{ selectedJob.desc || '-' }}</el-descriptions-item>
        <el-descriptions-item label="Cron表达式">{{ selectedJob.cron_expr }}</el-descriptions-item>
        <el-descriptions-item label="执行模式">
          <el-tag :type="getModeType(selectedJob.mode)">
            {{ getModeText(selectedJob.mode) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="执行命令">
          <pre>{{ selectedJob.command }}</pre>
        </el-descriptions-item>
        <el-descriptions-item label="任务状态">
          <el-tag :type="getStateType(selectedJob.state)">
            {{ getStateText(selectedJob.state) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="执行模式">
          {{ getAllowModeText(selectedJob.allow_mode) }}
        </el-descriptions-item>
        <el-descriptions-item label="最大执行次数">
          {{ selectedJob.max_run_count || '∞' }}
        </el-descriptions-item>
        <el-descriptions-item label="已执行次数">{{ selectedJob.run_count }}</el-descriptions-item>
        <el-descriptions-item label="创建时间">{{ formatDate(selectedJob.created_at) }}</el-descriptions-item>
        <el-descriptions-item label="更新时间">{{ formatDate(selectedJob.updated_at) }}</el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showDetailDialog = false">关闭</el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 任务日志对话框 -->
    <el-dialog
      v-model="showLogsDialog"
      title="任务执行日志"
      :width="isLogsFullscreen ? '100%' : '900px'"
      :fullscreen="isLogsFullscreen"
      :close-on-click-modal="false"
    >
      <template #header="{ close, titleId, titleClass }">
        <div class="el-dialog__header">
          <span :id="titleId" :class="titleClass">任务执行日志</span>
          <div class="el-dialog__headerbtn">
            <button
              type="button"
              class="el-dialog__headerbtn"
              :title="isLogsFullscreen ? '退出全屏' : '全屏显示'"
              @click="isLogsFullscreen = !isLogsFullscreen"
              style="margin-right: 40px;"
            >
              <el-icon>
                <FullScreen v-if="!isLogsFullscreen" />
                <Close v-else />
              </el-icon>
            </button>
            <button
              type="button"
              class="el-dialog__headerbtn"
              aria-label="Close"
              @click="close"
            >
              <el-icon><Close /></el-icon>
            </button>
          </div>
        </div>
      </template>
      <div v-loading="logsLoading">
        <el-row :gutter="20" style="margin-bottom: 20px;">
          <el-col :span="6">
            <el-date-picker
              v-model="logDate"
              type="date"
              placeholder="选择日期"
              @change="loadJobLogs"
            />
          </el-col>
          <el-col :span="6">
            <el-select v-model="logLimit" placeholder="显示条数" @change="loadJobLogs">
              <el-option label="10条" :value="10" />
              <el-option label="20条" :value="20" />
              <el-option label="50条" :value="50" />
              <el-option label="100条" :value="100" />
            </el-select>
          </el-col>
          <el-col :span="6">
            <el-select v-model="logSortOrder" placeholder="排序方式" @change="loadJobLogs">
              <el-option label="倒序显示" value="desc" />
              <el-option label="正序显示" value="asc" />
            </el-select>
          </el-col>
          <el-col :span="6" class="text-right">
            <el-button type="primary" @click="loadJobLogs">刷新</el-button>
            <el-button type="danger" @click="clearJobLogs">清除日志</el-button>
          </el-col>
        </el-row>
        
        <el-table
          :data="jobLogs" 
          style="width: 100%" 
          :max-height="isLogsFullscreen ? 'calc(100vh - 250px)' : '500'"
          v-loading="logsLoading"
        >
          <el-table-column label="序号" width="60" type="index">
            <template #default="scope">
              {{ 
                logSortOrder === 'asc' 
                  ?  scope.$index + 1 
                  : logLimit -scope.$index
              }}
            </template>
          </el-table-column>
          <el-table-column prop="time" label="执行时间" width="120" />
          <el-table-column prop="mode" label="执行模式" width="60" />
          <el-table-column label="状态" width="60">
            <template #default="scope">
              <el-tag :type="scope.row.status === '成功' ? 'success' : 'danger'">
                {{ scope.row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="duration_ms" label="耗时(ms)" width="80" />
          <el-table-column label="输出内容" min-width="300">
            <template #default="scope">
              <div class="log-output-container">
                <div class="log-output-preview" :class="{ 'expanded': expandedLogs.has(scope.$index) }">
                  <div v-if="scope.row.stdout" class="log-section">
                    <div class="log-section-title">
                      <span>输出</span>
                      <el-tag size="small" type="success">标准输出</el-tag>
                    </div>
                    <pre class="log-content stdout">{{ scope.row.stdout }}</pre>
                  </div>
                  <div v-if="scope.row.stderr" class="log-section">
                    <div class="log-section-title">
                      <span>错误</span>
                      <el-tag size="small" type="danger">标准错误</el-tag>
                    </div>
                    <pre class="log-content stderr">{{ scope.row.stderr }}</pre>
                  </div>
                  <div v-if="!scope.row.stdout && !scope.row.stderr" class="log-section">
                    <div class="log-section-title">
                      <span>信息</span>
                      <el-tag size="small" type="info">系统消息</el-tag>
                    </div>
                    <div class="log-content message">{{ scope.row.message || '无输出内容' }}</div>
                  </div>
                </div>
                <div class="log-actions">
                  <el-button 
                    size="small" 
                    type="text" 
                    @click="toggleLogExpand(scope.$index)"
                  >
                    {{ expandedLogs.has(scope.$index) ? '收起' : '展开' }}
                    <el-icon>
                      <ArrowDown v-if="!expandedLogs.has(scope.$index)" />
                      <ArrowUp v-else />
                    </el-icon>
                  </el-button>
                  <el-button 
                    size="small" 
                    type="text"
                    @click="copyLogContent(scope.row)"
                  >
                    复制
                    <el-icon><DocumentCopy /></el-icon>
                  </el-button>
                  <el-button 
                    size="small" 
                    type="text"
                    @click="showLogDetail(scope.row)"
                  >
                    详情
                    <el-icon><View /></el-icon>
                  </el-button>
                </div>
              </div>
            </template>
          </el-table-column>
        </el-table>
        
        <el-pagination
          v-if="logTotal > logLimit"
          :current-page="logPage"
          :page-size="logLimit"
          :total="logTotal"
          layout="prev, pager, next, jumper, total"
          @current-change="handleLogPageChange"
          style="margin-top: 20px; text-align: center;"
        />
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive, computed, watch } from 'vue'
import { jobApi } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'
import { FullScreen, Close, ArrowDown, ArrowUp, DocumentCopy, View } from '@element-plus/icons-vue'

// 数据
const loading = ref(false)
const jobList = ref([])
const searchParams = ref({
  name: '',
  state: '-1',
  mode: ''
})
const pagination = ref({
  page: 1,
  size: 10,
  total: 0
})

// 对话框控制
const showAddDialog = ref(false)
const showDetailDialog = ref(false)
const showLogsDialog = ref(false)
const showCronHelp = ref(false)
const isEdit = ref(false)
const submitLoading = ref(false)
const isLogsFullscreen = ref(false)
const expandedLogs = ref(new Set()) // 存储展开的日志ID

// 命令配置
const commandConfig = reactive({
  http: {
    url: '',
    mode: 'GET',
    headers: '',
    data: '',
    cookies: '',
    proxy: '',
    times: 0,
    interval: 0,
    result: '',
    timeout: 60
  },
  command: {
    command: '',
    workdir: '',
    env: '',
    times: 0,
    interval: 0,
    timeout: 30
  },
  func: {
    name: '',
    arg: '',
    times: 0,
    interval: 0
  }
})

// 函数列表
const functionList = ref([])
const selectedFunction = computed(() => {
  return functionList.value.find(f => f.name === commandConfig.func.name)
})

// 表单数据
const jobForm = reactive({
  id: '',
  name: '',
  desc: '',
  cron_expr: '',
  mode: 'http',
  command: '',
  state: 0,
  allow_mode: 0,
  max_run_count: 0
})

// 表单引用和验证规则
const jobFormRef = ref()
const jobRules = {
  name: [{ required: true, message: '请输入任务名称', trigger: 'blur' }],
  cron_expr: [{ required: true, message: '请输入Cron表达式', trigger: 'blur' }],
  mode: [{ required: true, message: '请选择执行模式', trigger: 'change' }],
  command: [{ required: true, message: '请输入执行内容', trigger: 'blur' }]
}

const selectedJob = ref(null)
const jobLogs = ref([])
const logsLoading = ref(false)
const logDate = ref(new Date())
const logLimit = ref(10)
const logPage = ref(1)
const logTotal = ref(0)
const logSortOrder = ref('desc') // 排序方式：desc-倒序，asc-正序

// 方法
const handleAddClick = () => {
  // 预加载函数列表，避免点击函数模式时空列表
  loadFunctions()
  showAddEditDialog()
}

const toggleLogExpand = (logId) => {
  if (expandedLogs.value.has(logId)) {
    expandedLogs.value.delete(logId)
  } else {
    expandedLogs.value.add(logId)
  }
}

const copyLogContent = async (log) => {
  let content = ''
  if (log.stdout) content += `输出:\n${log.stdout}\n\n`
  if (log.stderr) content += `错误:\n${log.stderr}\n\n`
  if (!log.stdout && !log.stderr && log.message) content += `信息:\n${log.message}`
  
  try {
    await navigator.clipboard.writeText(content.trim())
    ElMessage.success('日志内容已复制到剪贴板')
  } catch (error) {
    ElMessage.error('复制失败，请手动选择复制')
  }
}

const showLogDetail = (log) => {
  ElMessageBox.alert(
    `<div class="log-detail">
      <div class="log-detail-section">
        <h4>任务信息</h4>
        <p><strong>任务ID:</strong> ${log.job_id}</p>
        <p><strong>执行时间:</strong> ${formatDate(log.created_at)}</p>
        <p><strong>状态:</strong> ${log.status}</p>
        <p><strong>耗时:</strong> ${log.duration_ms}ms</p>
      </div>
      ${log.stdout ? `<div class="log-detail-section"><h4>标准输出</h4><pre>${log.stdout}</pre></div>` : ''}
      ${log.stderr ? `<div class="log-detail-section"><h4>标准错误</h4><pre>${log.stderr}</pre></div>` : ''}
      ${!log.stdout && !log.stderr && log.message ? `<div class="log-detail-section"><h4>系统消息</h4><pre>${log.message}</pre></div>` : ''}
    </div>`,
    '日志详情',
    {
      dangerouslyUseHTMLString: true,
      customClass: 'log-detail-dialog',
      confirmButtonText: '关闭',
      callback: () => {}
    }
  )
}

const loadJobs = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.value.page,
      size: pagination.value.size,
      ...searchParams.value
    }
    const res = await jobApi.getJobs(params)
    // 适配不同的API响应格式
    if (Array.isArray(res.data)) {
      // 格式: {data: [...], total: 1}
      jobList.value = res.data || []
      pagination.value.total = res.total || res.data.length || 0
    } else {
      // 格式: {data: {list: [...], total: 1}}
      jobList.value = res.data.list || []
      pagination.value.total = res.data.total || 0
    }
  } catch (error) {
    ElMessage.error('获取任务列表失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

const resetSearch = () => {
  searchParams.value = { name: '', state: '-1', mode: '' }
  pagination.value.page = 1
  loadJobs()
}

const handleSizeChange = (size) => {
  pagination.value.size = size
  loadJobs()
}

const handleCurrentChange = (page) => {
  pagination.value.page = page
  loadJobs()
}

const getModeText = (mode) => {
  const modeMap = {
    http: 'HTTP',
    command: '命令',
    func: '函数'
  }
  return modeMap[mode] || mode
}

const getModeType = (mode) => {
  const typeMap = {
    http: 'primary',
    command: 'success',
    func: 'warning'
  }
  return typeMap[mode] || 'info'
}

const getStateText = (state) => {
  const stateMap = {
    0: '等待',
    1: '运行中',
    2: '已停止'
  }
  return stateMap[state] || '未知'
}

const getStateType = (state) => {
  const typeMap = {
    0: 'warning',
    1: 'success',
    2: 'danger'
  }
  return typeMap[state] || 'info'
}

const getAllowModeText = (mode) => {
  const modeMap = {
    0: '并行执行',
    1: '串行执行（跳过）',
    2: '串行执行（排队）'
  }
  return modeMap[mode] || '未知'
}

const formatDate = (date) => {
  if (!date) return '-'
  return new Date(date).toLocaleString('zh-CN')
}

// 解析命令
const parseCommand = (command, mode) => {
  if (!command) return
  
  try {
    if (mode === 'http') {
      const lines = command.split('\n')
      const config = {
        url: '',
        mode: 'GET',
        headers: '',
        data: '',
        cookies: '',
        proxy: '',
        times: 0,
        interval: 0,
        result: '',
        timeout: 60
      }
      
      lines.forEach(line => {
        line = line.trim()
        if (line.startsWith('【url】')) {
          config.url = line.replace('【url】', '').trim()
        } else if (line.startsWith('【mode】')) {
          config.mode = line.replace('【mode】', '').trim()
        } else if (line.startsWith('【headers】')) {
          config.headers = line.replace('【headers】', '').trim()
        } else if (line.startsWith('【data】')) {
          config.data = line.replace('【data】', '').trim()
        } else if (line.startsWith('【cookies】')) {
          config.cookies = line.replace('【cookies】', '').trim()
        } else if (line.startsWith('【proxy】')) {
          config.proxy = line.replace('【proxy】', '').trim()
        } else if (line.startsWith('【times】')) {
          config.times = parseInt(line.replace('【times】', '').trim()) || 0
        } else if (line.startsWith('【interval】')) {
          config.interval = parseInt(line.replace('【interval】', '').trim()) || 0
        } else if (line.startsWith('【result】')) {
          config.result = line.replace('【result】', '').trim()
        } else if (line.startsWith('【timeout】')) {
          config.timeout = parseInt(line.replace('【timeout】', '').trim()) || 60
        }
      })
      
      Object.assign(commandConfig.http, config)
    } else if (mode === 'command') {
      const lines = command.split('\n')
      const config = {
        command: '',
        workdir: '',
        env: '',
        times: 0,
        interval: 0,
        timeout: 30
      }
      
      lines.forEach(line => {
        line = line.trim()
        if (line.startsWith('【command】')) {
          config.command = line.replace('【command】', '').trim()
        } else if (line.startsWith('【workdir】')) {
          config.workdir = line.replace('【workdir】', '').trim()
        } else if (line.startsWith('【env】')) {
          config.env = line.replace('【env】', '').trim()
        } else if (line.startsWith('【times】')) {
          config.times = parseInt(line.replace('【times】', '').trim()) || 0
        } else if (line.startsWith('【interval】')) {
          config.interval = parseInt(line.replace('【interval】', '').trim()) || 0
        } else if (line.startsWith('【timeout】')) {
          config.timeout = parseInt(line.replace('【timeout】', '').trim()) || 30
        }
      })
      
      Object.assign(commandConfig.command, config)
    } else if (mode === 'func') {
      const lines = command.split('\n')
      const config = {
        name: '',
        arg: '',
        times: 0,
        interval: 0
      }
      
      lines.forEach(line => {
        line = line.trim()
        if (line.startsWith('【name】')) {
          config.name = line.replace('【name】', '').trim()
        } else if (line.startsWith('【arg】')) {
          config.arg = line.replace('【arg】', '').trim()
        } else if (line.startsWith('【times】')) {
          config.times = parseInt(line.replace('【times】', '').trim()) || 0
        } else if (line.startsWith('【interval】')) {
          config.interval = parseInt(line.replace('【interval】', '').trim()) || 0
        }
      })
      
      Object.assign(commandConfig.func, config)
    }
  } catch (error) {
    console.error('解析命令失败:', error)
  }
}

// 构建命令
const buildCommand = (mode) => {
  let command = ''
  
  if (mode === 'http') {
    const config = commandConfig.http
    command = `【url】${config.url}`
    if (config.mode && config.mode !== 'GET') {
      command += `\n【mode】${config.mode}`
    }
    if (config.headers) {
      command += `\n【headers】${config.headers}`
    }
    if (config.data) {
      command += `\n【data】${config.data}`
    }
    if (config.cookies) {
      command += `\n【cookies】${config.cookies}`
    }
    if (config.proxy) {
      command += `\n【proxy】${config.proxy}`
    }
    if (config.times > 0) {
      command += `\n【times】${config.times}`
    }
    if (config.interval > 0) {
      command += `\n【interval】${config.interval}`
    }
    if (config.result) {
      command += `\n【result】${config.result}`
    }
    if (config.timeout !== 60) {
      command += `\n【timeout】${config.timeout}`
    }
  } else if (mode === 'command') {
    const config = commandConfig.command
    command = `【command】${config.command}`
    if (config.workdir) {
      command += `\n【workdir】${config.workdir}`
    }
    if (config.env) {
      command += `\n【env】${config.env}`
    }
    if (config.times > 0) {
      command += `\n【times】${config.times}`
    }
    if (config.interval > 0) {
      command += `\n【interval】${config.interval}`
    }
    if (config.timeout !== 30) {
      command += `\n【timeout】${config.timeout}`
    }
  } else if (mode === 'func') {
    const config = commandConfig.func
    command = `【name】${config.name}`
    if (config.arg) {
      command += `\n【arg】${config.arg}`
    }
    if (config.times > 0) {
      command += `\n【times】${config.times}`
    }
    if (config.interval > 0) {
      command += `\n【interval】${config.interval}`
    }
  }
  
  return command
}

const handleModeChange = (mode) => {
  // 清空之前的配置
  if (mode === 'http') {
    Object.assign(commandConfig.http, {
      url: '',
      mode: 'GET',
      headers: '',
      data: '',
      cookies: '',
      proxy: '',
      times: 0,
      interval: 0,
      result: '',
      timeout: 60
    })
  } else if (mode === 'command') {
    Object.assign(commandConfig.command, {
      command: '',
      workdir: '',
      env: '',
      times: 0,
      interval: 0,
      timeout: 30
    })
  } else if (mode === 'func') {
    Object.assign(commandConfig.func, {
      name: '',
      arg: '',
      times: 0,
      interval: 0
    })
  }
}

const loadFunctions = async () => {
  try {
    const res = await jobApi.getFunctions()
    functionList.value = res.data || []
  } catch (error) {
    ElMessage.error('获取函数列表失败: ' + error.message)
  }
}

const showAddEditDialog = (job = null) => {
  isEdit.value = !!job
  
  if (job) {
    Object.assign(jobForm, {
      id: job.id || '',
      name: job.name || '',
      desc: job.desc || '',
      cron_expr: job.cron_expr || '',
      mode: job.mode || 'http',
      command: job.command || '',
      state: job.state || 0,
      allow_mode: job.allow_mode || 0,
      max_run_count: job.max_run_count || 0
    })
    // 解析现有命令
    parseCommand(job.command, job.mode)
  } else {
    Object.assign(jobForm, {
      name: '',
      desc: '',
      cron_expr: '',
      mode: 'http',
      command: '',
      state: 0,
      allow_mode: 0,
      max_run_count: 0
    })
    handleModeChange('http')
  }
  
  showAddDialog.value = true
}

const submitJob = async () => {
  try {
    // 根据模式设置command字段用于验证
    if (jobForm.mode === 'http') {
      jobForm.command = commandConfig.http.url
    } else if (jobForm.mode === 'command') {
      jobForm.command = commandConfig.command.command
    } else if (jobForm.mode === 'func') {
      jobForm.command = commandConfig.func.name
    }

    // 表单验证
    if (!jobFormRef.value) {
      // 如果表单引用未定义，使用手动验证
      if (!jobForm.name) {
        ElMessage.error('请输入任务名称')
        return
      }
      if (!jobForm.cron_expr) {
        ElMessage.error('请输入Cron表达式')
        return
      }
      if (!jobForm.mode) {
        ElMessage.error('请选择执行模式')
        return
      }
      if (jobForm.mode === 'http' && !commandConfig.http.url) {
        ElMessage.error('请输入URL地址')
        return
      }
      if (jobForm.mode === 'command' && !commandConfig.command.command) {
        ElMessage.error('请输入执行命令')
        return
      }
      if (jobForm.mode === 'func' && !commandConfig.func.name) {
        ElMessage.error('请选择函数')
        return
      }
    } else {
      // 使用表单验证
      const valid = await jobFormRef.value.validate()
      if (!valid) {
        return
      }
    }
    
    // 验证具体模式的必填字段
    if (jobForm.mode === 'http' && !commandConfig.http.url) {
      ElMessage.error('请输入URL地址')
      return
    }
    if (jobForm.mode === 'command' && !commandConfig.command.command) {
      ElMessage.error('请输入执行命令')
      return
    }
    if (jobForm.mode === 'func' && !commandConfig.func.name) {
      ElMessage.error('请选择函数')
      return
    }
    
    // 构建命令
    jobForm.command = buildCommand(jobForm.mode)
    
    submitLoading.value = true
    
    if (isEdit.value) {
      await jobApi.editJob(jobForm)
      ElMessage.success('任务更新成功')
    } else {
      // 新增任务时不发送id字段
      const { id, ...addData } = jobForm
      await jobApi.addJob(addData)
      ElMessage.success('任务创建成功')
    }
    
    showAddDialog.value = false
    loadJobs()
  } catch (error) {
    ElMessage.error(isEdit.value ? '任务更新失败: ' : '任务创建失败: ' + error.message)
  } finally {
    submitLoading.value = false
  }
}

const viewJobDetail = (job) => {
  selectedJob.value = job
  showDetailDialog.value = true
}

const toggleJobState = async (job_id,type) => {
  try {
    if (type === 1) {
      await jobApi.runJob(job_id)
      ElMessage.success('任务已启动')
    } else {
      await jobApi.stopJob(job_id)
      ElMessage.success('任务已停止')
    }
    loadJobs()
  } catch (error) {
    ElMessage.error('操作失败: ' + error.message)
  }
}



const handleCommand = (command, job) => {
  switch (command) {
    case 'edit':
      showAddEditDialog(job)
      break
    case 'logs':
      viewJobLogs(job)
      break
    case 'delete':
      deleteJob(job)
      break
     case 'stop':
        toggleJobState(job.id,2)
      break
    case 'restart':
      restartJob(job.id)
      break
  }
}

const deleteJob = async (job) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除任务 "${job.name}" 吗？此操作不可恢复！`,
      '确认删除',
      { type: 'warning' }
    )
    await jobApi.deleteJob(job.id)
    ElMessage.success('任务删除成功')
    loadJobs()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败: ' + error.message)
    }
  }
}
const restartJob = async (job_id) => {
  try {
    await jobApi.restartJob(job_id)
    ElMessage.success('任务重启成功')
    loadJobs()
  } catch (error) {
    ElMessage.error('重启失败: ' + error.message)
  }
}

const viewJobLogs = (job) => {
  selectedJob.value = job
  logDate.value = new Date()
  logPage.value = 1
  logLimit.value = 10
  logSortOrder.value = 'desc' // 默认倒序
  logTotal.value = 0
  showLogsDialog.value = true
  loadJobLogs()
}

// 获取本地日期字符串，避免UTC转换问题
const getLocalDateString = (date) => {
  if (!date) return ''
  const d = new Date(date)
  const year = d.getFullYear()
  const month = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const loadJobLogs = async () => {
  logsLoading.value = true
  try {
    
    const params = {
      id: selectedJob.value.id,
      date: logDate.value ? getLocalDateString(logDate.value) : '',
      limit: logLimit.value,
      page: logPage.value
    }
    const res = await jobApi.getJobLogs(params)
    let logs = res.data || []
    logTotal.value = res.total || 0
    
    // 前端排序：根据执行时间排序
    logs.sort((a, b) => {
      const timeA = new Date(a.time || 0)
      const timeB = new Date(b.time || 0)
      return logSortOrder.value === 'asc' ? timeA - timeB : timeB - timeA
    })
    
    jobLogs.value = logs
  } catch (error) {
    ElMessage.error('获取日志失败: ' + error.message)
  } finally {
    logsLoading.value = false
  }
}

const handleLogPageChange = (page) => {
  logPage.value = page
  loadJobLogs()
}

const clearJobLogs = async () => {
  try {
    const targetDate = logDate.value ? getLocalDateString(logDate.value) : getLocalDateString(new Date())
    await ElMessageBox.confirm(
      `确定要清除 ${targetDate} 的日志吗？`,
      '确认清除日志'
    )
    await jobApi.clearLogs({
      id: selectedJob.value.id,
      date: targetDate,
      type: 'job'
    })
    ElMessage.success(`${targetDate} 的日志已清除`)
    loadJobLogs()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('清除失败: ' + error.message)
    }
  }
}

const runAllJobs = async () => {
  try {
    await jobApi.runAllJobs()
    ElMessage.success('所有任务已启动')
    loadJobs()
  } catch (error) {
    ElMessage.error('启动失败: ' + error.message)
  }
}

const stopAllJobs = async () => {
  try {
    await jobApi.stopAllJobs()
    ElMessage.success('所有任务已停止')
    loadJobs()
  } catch (error) {
    ElMessage.error('停止失败: ' + error.message)
  }
}

const calibrateJobs = async () => {
  try {
    await jobApi.calibrateJobs()
    ElMessage.success('任务校准完成')
    loadJobs()
  } catch (error) {
    ElMessage.error('校准失败: ' + error.message)
  }
}

// 初始化
onMounted(() => {
  loadJobs()
})
</script>

<style scoped>
.jobs-page {
  padding: 20px;
  background-color: #f5f7fa;
}

.page-header {
  margin-bottom: 20px;
  align-items: center;
  background: #ffffff;
  padding: 16px;
  border-radius: 4px;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.08);
}

.text-right {
  text-align: right;
}

.search-card {
  margin-bottom: 20px;
  border-radius: 4px;
}

.search-card :deep(.el-card__body) {
  padding: 16px;
}

.pagination {
  margin-top: 20px;
  text-align: right;
  padding: 16px 0;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  background: #f8f8f8;
  padding: 8px;
  border-radius: 4px;
}

:deep(.el-table) {
  border-radius: 4px;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.08);
  border: 1px solid #ebeef5;
}

:deep(.el-table__header-wrapper) {
  border-radius: 4px 4px 0 0;
}

:deep(.el-table th) {
  background-color: #f5f7fa;
  color: #606266;
  font-weight: 600;
}

:deep(.el-table--border td) {
  border-right: 1px solid #ebeef5;
}

:deep(.el-table--border th) {
  border-right: 1px solid #ebeef5;
}

:deep(.el-table .el-table__row:hover > td) {
  background-color: #f5f7fa;
  transition: background-color 0.3s ease;
}

:deep(.el-pagination) {
  background: #ffffff;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  padding: 8px 16px;
}

:deep(.el-pagination .btn-prev),
:deep(.el-pagination .btn-next) {
  background: #ffffff;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  margin-right: 8px;
}

:deep(.el-pagination .el-pager li) {
  background: #ffffff;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  margin-right: 8px;
}

:deep(.el-pagination .el-pager li.active) {
  background: #409eff;
  border-color: #409eff;
  color: #ffffff;
}

:deep(.el-pagination .el-select .el-input) {
  margin-right: 8px;
}

:deep(.el-pagination__jump) {
  margin-left: 8px;
}

/* 日志输出样式 */
.log-output-container {
  position: relative;
}

.log-output-preview {
  max-height: 120px;
  overflow: hidden;
  transition: max-height 0.3s ease;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  background: #fafafa;
}

.log-output-preview.expanded {
  max-height: none;
  overflow: visible;
}

.log-section {
  margin-bottom: 8px;
  padding: 8px;
  border-bottom: 1px solid #eee;
}

.log-section:last-child {
  border-bottom: none;
  margin-bottom: 0;
}

.log-section-title {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
  font-weight: 600;
  font-size: 12px;
}

.log-content {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: 'Consolas', 'Monaco', 'Lucida Console', monospace;
  font-size: 11px;
  line-height: 1.4;
  color: #333;
  background: #fff;
  padding: 6px 8px;
  border-radius: 3px;
  border-left: 3px solid #ddd;
}

.log-content.stdout {
  border-left-color: #67c23a;
  background: #f0f9ff;
}

.log-content.stderr {
  border-left-color: #f56c6c;
  background: #fef0f0;
}

.log-content.message {
  border-left-color: #909399;
  background: #f4f4f5;
  color: #606266;
}

.log-actions {
  margin-top: 8px;
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.log-actions .el-button {
  padding: 2px 8px;
  font-size: 11px;
}

/* 日志详情弹窗样式 */
:deep(.log-detail-dialog) {
  width: 80%;
  max-width: 800px;
}

:deep(.log-detail) {
  max-height: 600px;
  overflow-y: auto;
}

:deep(.log-detail-section) {
  margin-bottom: 20px;
}

:deep(.log-detail-section h4) {
  margin: 0 0 10px 0;
  color: #303133;
  font-size: 14px;
  font-weight: 600;
}

:deep(.log-detail-section pre) {
  margin: 0;
  padding: 12px;
  background: #f5f7fa;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: 'Consolas', 'Monaco', 'Lucida Console', monospace;
  font-size: 12px;
  line-height: 1.5;
  color: #303133;
}
</style>
