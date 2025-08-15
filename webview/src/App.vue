<template>
  <div id="app">
    <el-container class="layout-container">
      <el-aside width="200px" class="layout-aside">
        <div class="logo">
          <h3>任务调度系统</h3>
        </div>
        <el-menu
          :default-active="$route.path"
          class="layout-menu"
          router
          background-color="#304156"
          text-color="#bfcbd9"
          active-text-color="#409EFF"
        >
          <el-menu-item index="/">
            <el-icon><House /></el-icon>
            <span>首页</span>
          </el-menu-item>
          <el-menu-item index="/jobs">
            <el-icon><Clock /></el-icon>
            <span>任务管理</span>
          </el-menu-item>
          <el-menu-item index="/settings">
            <el-icon><Setting /></el-icon>
            <span>系统设置</span>
          </el-menu-item>
        </el-menu>
      </el-aside>
      
      <el-container>
        <el-header class="layout-header">
          <div class="header-content">
            <h2>{{ $route.meta.title || '任务调度系统' }}</h2>
          </div>
        </el-header>
        
        <el-main class="layout-main">
          <router-view />
        </el-main>
      </el-container>
    </el-container>
  </div>
</template>

<script setup>
import { House, Clock, Setting } from '@element-plus/icons-vue'
</script>

<style scoped>
.layout-container {
  height: 100vh;
}

.layout-aside {
  background-color: #304156;
}

.logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #2b3a4b;
  color: white;
}

.logo h3 {
  margin: 0;
  font-size: 16px;
}

.layout-menu {
  border-right: none;
}

.layout-header {
  background-color: #fff;
  box-shadow: 0 1px 4px rgba(0,21,41,.08);
  display: flex;
  align-items: center;
}

.header-content h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 500;
}

.layout-main {
  background-color: #f5f5f5;
  padding: 20px;
}
</style>