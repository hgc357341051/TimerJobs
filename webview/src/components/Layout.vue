<template>
  <el-container class="layout-container">
    <!-- 侧边栏 -->
    <el-aside width="200px" class="sidebar">
      <div class="logo">
        <el-icon><Setting /></el-icon>
        <span>任务调度系统</span>
      </div>
      <el-menu
        :default-active="activeMenu"
        router
        class="menu"
      >
        <el-menu-item index="/">
          <el-icon><Monitor /></el-icon>
          <span>系统概览</span>
        </el-menu-item>
        <el-menu-item index="/jobs">
          <el-icon><Calendar /></el-icon>
          <span>任务管理</span>
        </el-menu-item>
        <el-menu-item index="/logs">
          <el-icon><Document /></el-icon>
          <span>系统日志</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <!-- 主内容区 -->
    <el-container>
      <!-- 顶部栏 -->
      <el-header class="header">
        <div class="header-content">
          <div class="breadcrumb">
            <el-breadcrumb separator="/">
              <el-breadcrumb-item>{{ $route.meta.title || '任务调度系统' }}</el-breadcrumb-item>
            </el-breadcrumb>
          </div>
          <div class="header-actions">
            <el-button @click="refreshData" :loading="refreshing">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
            <el-dropdown @command="handleCommand">
              <el-button>
                <el-icon><User /></el-icon>
                用户
                <el-icon><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="logout">退出登录</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </el-header>

      <!-- 主要内容 -->
      <el-main class="main-content">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'

const route = useRoute()
const router = useRouter()

const refreshing = ref(false)

const activeMenu = computed(() => route.path)

const refreshData = () => {
  refreshing.value = true
  // 触发子组件刷新
  window.dispatchEvent(new CustomEvent('refresh-data'))
  setTimeout(() => {
    refreshing.value = false
    ElMessage.success('数据已刷新')
  }, 1000)
}

const handleCommand = (command) => {
  switch (command) {
    case 'logout':
      // 这里可以添加退出登录逻辑
      ElMessage.info('退出登录功能待实现')
      break
  }
}
</script>

<style scoped>
.layout-container {
  height: 100vh;
  margin: 0;
  padding: 0;
  overflow: hidden;
}

.sidebar {
  background: #fff;
  border-right: 1px solid #e4e7ed;
  box-shadow: none;
  margin: 0;
  padding: 0;
}

.logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  font-weight: bold;
  color: #409eff;
  border-bottom: 1px solid #e4e7ed;
  margin: 0;
  padding: 0;
}

.logo .el-icon {
  margin-right: 8px;
  font-size: 20px;
}

.menu {
  border-right: none;
  box-shadow: none;
  margin: 0;
  padding: 0;
}

.header {
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
  padding: 0 20px;
  margin: 0;
  box-shadow: none;
}

.header-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 100%;
  margin: 0;
  padding: 0;
}

.breadcrumb {
  font-size: 14px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.main-content {
  background: #f5f7fa;
  padding: 20px;
  overflow-y: auto;
  margin: 0;
  border: none;
  box-shadow: none;
}

/* 确保没有意外的边距和边框 */
:deep(.el-container) {
  margin: 0 !important;
  padding: 0 !important;
  border: none !important;
  box-shadow: none !important;
}

:deep(.el-aside) {
  box-shadow: none !important;
  border: none !important;
  margin: 0 !important;
  padding: 0 !important;
}

:deep(.el-header) {
  box-shadow: none !important;
  border: none !important;
  margin: 0 !important;
  padding: 0 !important;
}

:deep(.el-main) {
  box-shadow: none !important;
  border: none !important;
  margin: 0 !important;
  padding: 0 !important;
}

/* 移除所有可能的重复边框 */
:deep(.el-card) {
  border: 1px solid #e4e7ed !important;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1) !important;
}
</style>