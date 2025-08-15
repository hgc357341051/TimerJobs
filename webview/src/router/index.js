import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    name: 'Home',
    component: () => import('@/views/Home.vue'),
    meta: { title: '系统概览' }
  },
  {
    path: '/jobs',
    name: 'Jobs',
    component: () => import('@/views/Jobs.vue'),
    meta: { title: '任务管理' }
  },
  {
    path: '/jobs/:id',
    name: 'JobDetail',
    component: () => import('@/views/JobDetail.vue'),
    meta: { title: '任务详情' }
  },
  {
    path: '/settings',
    name: 'Settings',
    component: () => import('@/views/Settings.vue'),
    meta: { title: '系统设置' }
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

export default router