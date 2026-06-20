<template>
  <el-container class="layout">
    <el-aside width="220px" class="aside">
      <div class="brand">DigitarloPanel</div>
      <el-menu :default-active="activeRoute" router class="menu">
        <el-menu-item index="/dashboard"><el-icon><Odometer /></el-icon><span>Dashboard</span></el-menu-item>
        <el-menu-item index="/assistant"><el-icon><ChatDotRound /></el-icon><span>KI-Assistent</span></el-menu-item>
        <el-menu-item index="/files"><el-icon><Folder /></el-icon><span>Dateien</span></el-menu-item>
        <el-menu-item index="/software"><el-icon><Box /></el-icon><span>Software</span></el-menu-item>
        <el-menu-item index="/services"><el-icon><Operation /></el-icon><span>Dienste</span></el-menu-item>
        <el-menu-item index="/websites"><el-icon><Link /></el-icon><span>Websites</span></el-menu-item>
        <el-menu-item index="/databases"><el-icon><Coin /></el-icon><span>Datenbanken</span></el-menu-item>
        <el-menu-item index="/ssl"><el-icon><Key /></el-icon><span>SSL</span></el-menu-item>
        <el-menu-item index="/cron"><el-icon><Timer /></el-icon><span>Cron-Jobs</span></el-menu-item>
        <el-menu-item index="/docker"><el-icon><Ship /></el-icon><span>Docker</span></el-menu-item>
        <el-menu-item index="/compose"><el-icon><Files /></el-icon><span>App-Stacks</span></el-menu-item>
        <el-menu-item index="/ftp"><el-icon><Upload /></el-icon><span>FTP</span></el-menu-item>
        <el-menu-item index="/dns"><el-icon><Connection /></el-icon><span>DNS</span></el-menu-item>
        <el-menu-item index="/mail"><el-icon><Message /></el-icon><span>Mail</span></el-menu-item>
        <el-menu-item index="/firewall"><el-icon><Lock /></el-icon><span>Firewall</span></el-menu-item>
        <el-menu-item index="/backups"><el-icon><FolderChecked /></el-icon><span>Backups</span></el-menu-item>
        <el-menu-item index="/logs"><el-icon><Tickets /></el-icon><span>Logs</span></el-menu-item>
        <el-menu-item v-if="auth.isAdmin" index="/terminal"><el-icon><Monitor /></el-icon><span>Terminal</span></el-menu-item>
        <el-menu-item v-if="auth.isAdmin" index="/users"><el-icon><User /></el-icon><span>Benutzer</span></el-menu-item>
        <el-menu-item index="/audit"><el-icon><Document /></el-icon><span>Audit-Log</span></el-menu-item>
        <el-menu-item index="/settings"><el-icon><Setting /></el-icon><span>Einstellungen</span></el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <div class="spacer" />
        <el-dropdown @command="onCommand">
          <span class="user">
            <el-icon><UserFilled /></el-icon>
            {{ auth.user?.username || 'admin' }}
            <el-icon><ArrowDown /></el-icon>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="settings">Einstellungen</el-dropdown-item>
              <el-dropdown-item command="logout" divided>Abmelden</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </el-header>
      <el-main class="main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../store/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const activeRoute = computed(() => route.path)

onMounted(() => {
  if (!auth.user) auth.fetchMe().catch(() => {})
})

function onCommand(command: string) {
  if (command === 'logout') {
    auth.logout()
    router.push({ name: 'login' })
  } else if (command === 'settings') {
    router.push({ name: 'settings' })
  }
}
</script>

<style scoped>
.layout {
  height: 100%;
}
.aside {
  background: #1f2937;
  color: #fff;
}
.brand {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 18px;
  color: #409eff;
}
.menu {
  border-right: none;
  background: transparent;
}
.menu :deep(.el-menu-item) {
  color: #cbd5e1;
}
.menu :deep(.el-menu-item.is-active) {
  background: #111827;
  color: #409eff;
}
.header {
  display: flex;
  align-items: center;
  border-bottom: 1px solid #e4e7ed;
  background: #fff;
}
.spacer {
  flex: 1;
}
.user {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  outline: none;
}
.main {
  background: #f5f7fa;
}
</style>
