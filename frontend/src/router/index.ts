import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { getToken } from '../api/client'

const routes: RouteRecordRaw[] = [
  { path: '/login', name: 'login', component: () => import('../views/Login.vue') },
  {
    path: '/',
    component: () => import('../layout/MainLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      { path: '', redirect: '/dashboard' },
      { path: 'dashboard', name: 'dashboard', component: () => import('../views/Dashboard.vue') },
      { path: 'files', name: 'files', component: () => import('../views/Files.vue') },
      { path: 'software', name: 'software', component: () => import('../views/Software.vue') },
      { path: 'services', name: 'services', component: () => import('../views/Services.vue') },
      { path: 'websites', name: 'websites', component: () => import('../views/Websites.vue') },
      { path: 'databases', name: 'databases', component: () => import('../views/Databases.vue') },
      { path: 'ssl', name: 'ssl', component: () => import('../views/SSL.vue') },
      { path: 'cron', name: 'cron', component: () => import('../views/Cron.vue') },
      { path: 'docker', name: 'docker', component: () => import('../views/Docker.vue') },
      { path: 'firewall', name: 'firewall', component: () => import('../views/Firewall.vue') },
      { path: 'terminal', name: 'terminal', component: () => import('../views/Terminal.vue') },
      { path: 'backups', name: 'backups', component: () => import('../views/Backups.vue') },
      { path: 'logs', name: 'logs', component: () => import('../views/Logs.vue') },
      { path: 'users', name: 'users', component: () => import('../views/Users.vue') },
      { path: 'audit', name: 'audit', component: () => import('../views/Audit.vue') },
      { path: 'settings', name: 'settings', component: () => import('../views/Settings.vue') },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to) => {
  if (to.meta.requiresAuth && !getToken()) {
    return { name: 'login' }
  }
  if (to.name === 'login' && getToken()) {
    return { name: 'dashboard' }
  }
  return true
})

export default router
