<template>
  <div class="jobs">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>任务管理</span>
          <div class="header-actions">
            <el-button type="primary" @click="showAddDialog = true">
              <el-icon><Plus /></el-icon>
              添加任务
            </el-button>
            <el-button @click="refreshData" :loading="loading">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </div>
        </div>
      </template>

      <el-table :data="jobList" style="width: 100%" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="任务名称" />
        <el-table-column prop="desc" label="描述" show-overflow-tooltip />
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
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="scope">
            <el-button 
              link 
              type="primary" 
              @click="viewJobDetail(scope.row)"
            >
              详情
            </el-button>
            <el-button 
              link 
              type="success" 
              @click="runJob(scope.row)"
              :disabled="scope.row.state === 1"
            >
              运行
            </el-button>
            <el-button 
              link 
              type="warning" 
              @click="stopJob(scope.row)"
              :disabled="scope.row.state !== 1"
            >
              停止
            </el-button>
            <el-button 
              link 
              type="primary" 
              @click="editJob(scope.row)"
            >
              编辑
            </el-button>
            <el-button 
              link 
              type="danger" 
              @click="deleteJob(scope.row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.current"
          v-model:page-size="pagination.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 添加任务对话框 -->
    <el-dialog
      v-model="showAddDialog"
      title="添加任务"
      width="500px"
    >
      <el-form :model="jobForm" label-width="80px" ref="addFormRef" :rules="rules">
        <el-form-item label="任务名称" prop="name">
          <el-input v-model="jobForm.name" placeholder="请输入任务名称" />
        </el-form-item>
        <el-form-item label="任务描述" prop="desc">
          <el-input 
            v-model="jobForm.desc" 
            type="textarea" 
            :rows="3"
            placeholder="请输入任务描述"
          />
        </el-form-item>
        <el-form-item label="执行规则" prop="cron_expr">
          <el-input v-model="jobForm.cron_expr" placeholder="例如: */5 * * * * *" />
        </el-form-item>
        <el-form-item label="执行模式" prop="mode">
          <el-select v-model="jobForm.mode" placeholder="请选择执行模式">
            <el-option label="命令行" value="command" />
            <el-option label="HTTP请求" value="http" />
            <el-option label="函数调用" value="function" />
          </el-select>
        </el-form-item>
        <el-form-item label="执行内容" prop="command">
          <el-input 
            v-model="jobForm.command" 
            type="textarea" 
            :rows="4"
            placeholder="请输入执行内容"
          />
        </el-form-item>
        <el-form-item label="状态" prop="state">
          <el-radio-group v-model="jobForm.state">
            <el-radio :label="0">停止</el-radio>
            <el-radio :label="1">运行</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddDialog = false">取消</el-button>
        <el-button type="primary" @click="submitAdd" :loading="submitLoading">
          确认
        </el-button>
      </template>
    </el-dialog>

    <!-- 编辑任务对话框 -->
    <el-dialog
      v-model="showEditDialog"
      title="编辑任务"
      width="500px"
    >
      <el-form :model="editForm" label-width="80px" ref="editFormRef" :rules="rules">
        <el-form-item label="任务名称" prop="name">
          <el-input v-model="editForm.name" placeholder="请输入任务名称" />
        </el-form-item>
        <el-form-item label="任务描述" prop="desc">
          <el-input 
            v-model="editForm.desc" 
            type="textarea" 
            :rows="3"
            placeholder="请输入任务描述"
          />
        </el-form-item>
        <el-form-item label="执行规则" prop="cron_expr">
          <el-input v-model="editForm.cron_expr" placeholder="例如: */5 * * * * *" />
        </el-form-item>
        <el-form-item label="执行模式" prop="mode">
          <el-select v-model="editForm.mode" placeholder="请选择执行模式">
            <el-option label="命令行" value="command" />
            <el-option label="HTTP请求" value="http" />
            <el-option label="函数调用" value="function" />
          </el-select>
        </el-form-item>
        <el-form-item label="执行内容" prop="command">
          <el-input 
            v-model="editForm.command" 
            type="textarea" 
            :rows="4"
            placeholder="请输入执行内容"
          />
        </el-form-item>
        <el-form-item label="状态" prop="state">
          <el-radio-group v-model="editForm.state">
            <el-radio :label="0">停止</el-radio>
            <el-radio :label="1">运行</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditDialog = false">取消</el-button>
        <el-button type="primary" @click="submitEdit" :loading="submitLoading">
          确认
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { jobApi } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'

const router = useRouter()

const jobList = ref([])
const loading = ref(false)
const showAddDialog = ref(false)
const showEditDialog = ref(false)
const submitLoading = ref(false)

const pagination = ref({
  current: 1,
  pageSize: 10,
  total: 0
})

const jobForm = ref({
  name: '',
  desc: '',
  cron_expr: '',
  mode: 'command',
  command: '',
  state: 0
})

const editForm = ref({
  id: null,
  name: '',
  desc: '',
  cron_expr: '',
  mode: 'command',
  command: '',
  state: 0
})

const rules = {
  name: [{ required: true, message: '请输入任务名称', trigger: 'blur' }],
  cron_expr: [{ required: true, message: '请输入执行规则', trigger: 'blur' }],
  mode: [{ required: true, message: '请选择执行模式', trigger: 'change' }],
  command: [{ required: true, message: '请输入执行内容', trigger: 'blur' }]
}

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

const loadData = async () => {
  loading.value = true
  try {
    const response = await jobApi.getJobs({
      page: pagination.value.current,
      size: pagination.value.pageSize
    })
    
    jobList.value = response.data.list || []
    pagination.value.total = response.data.total || 0
  } catch (error) {
    ElMessage.error('加载任务列表失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

const handleSizeChange = (size) => {
  pagination.value.pageSize = size
  loadData()
}

const handleCurrentChange = (current) => {
  pagination.value.current = current
  loadData()
}

const refreshData = () => {
  loadData()
}

const viewJobDetail = (job) => {
  router.push(`/jobs/${job.id}`)
}

const runJob = async (job) => {
  try {
    await ElMessageBox.confirm('确定要运行此任务吗？', '提示', {
      type: 'warning'
    })
    
    await jobApi.runJob(job.id)
    ElMessage.success('任务已开始运行')
    loadData()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('运行任务失败: ' + error.message)
    }
  }
}

const stopJob = async (job) => {
  try {
    await ElMessageBox.confirm('确定要停止此任务吗？', '提示', {
      type: 'warning'
    })
    
    await jobApi.stopJob(job.id)
    ElMessage.success('任务已停止')
    loadData()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('停止任务失败: ' + error.message)
    }
  }
}

const editJob = (job) => {
  editForm.value = {
    id: job.id,
    name: job.name,
    desc: job.desc || '',
    cron_expr: job.cron_expr,
    mode: job.mode,
    command: job.command,
    state: job.state
  }
  showEditDialog.value = true
}

const deleteJob = async (job) => {
  try {
    await ElMessageBox.confirm('确定要删除此任务吗？此操作不可恢复！', '警告', {
      type: 'error'
    })
    
    await jobApi.deleteJob(job.id)
    ElMessage.success('任务已删除')
    loadData()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除任务失败: ' + error.message)
    }
  }
}

const submitAdd = async () => {
  submitLoading.value = true
  try {
    await jobApi.addJob(jobForm.value)
    ElMessage.success('任务添加成功')
    showAddDialog.value = false
    loadData()
    
    // 重置表单
    jobForm.value = {
      name: '',
      desc: '',
      cron_expr: '',
      mode: 'command',
      command: '',
      state: 0
    }
  } catch (error) {
    ElMessage.error('添加任务失败: ' + error.message)
  } finally {
    submitLoading.value = false
  }
}

const submitEdit = async () => {
  submitLoading.value = true
  try {
    await jobApi.editJob(editForm.value)
    ElMessage.success('任务更新成功')
    showEditDialog.value = false
    loadData()
  } catch (error) {
    ElMessage.error('更新任务失败: ' + error.message)
  } finally {
    submitLoading.value = false
  }
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.jobs {
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

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>