<template>
  <div>
    <h2 class="page-title">App-Store</h2>
    <el-alert v-if="data && !data.available" :closable="false" type="warning" show-icon style="margin-bottom: 16px">
      <div style="display: flex; align-items: center; gap: 12px">
        <span>Docker wird für den App-Store benötigt, ist aber nicht installiert.</span>
        <el-button type="primary" size="small" :loading="installingDocker" @click="installDocker">
          Docker installieren
        </el-button>
      </div>
    </el-alert>

    <div class="toolbar">
      <el-input v-model="search" placeholder="Apps durchsuchen…" style="width: 280px" clearable />
      <el-button @click="load">Aktualisieren</el-button>
    </div>

    <div v-for="group in filteredGroups" :key="group.category" style="margin-bottom: 24px">
      <h3 class="cat-title">{{ group.category }}</h3>
      <el-row :gutter="16">
        <el-col v-for="app in group.apps" :key="app.key" :span="8" style="margin-bottom: 16px">
          <el-card shadow="never" class="app-card">
            <div class="app-head">
              <b>{{ app.name }}</b>
              <el-tag v-if="app.installed" type="success" size="small">installiert</el-tag>
            </div>
            <p class="app-desc">{{ app.description }}</p>
            <div class="app-foot">
              <span class="port">Port {{ app.port }}</span>
              <el-button
                v-if="app.installed"
                size="small"
                @click="openApp(app)"
              >Öffnen</el-button>
              <el-button
                v-else
                type="primary"
                size="small"
                :loading="installing === app.key"
                :disabled="!data?.available"
                @click="install(app)"
              >Installieren</el-button>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/client'

const data = ref<any>(null)
const search = ref('')
const installing = ref('')
const installingDocker = ref(false)

async function installDocker() {
  installingDocker.value = true
  try {
    ElMessage.info('Docker wird installiert … (kann ein bis zwei Minuten dauern)')
    await http.post('/docker/install')
    ElMessage.success('Docker installiert')
    await load()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || 'Docker-Installation fehlgeschlagen')
  } finally {
    installingDocker.value = false
  }
}

const filteredGroups = computed(() => {
  if (!data.value) return []
  const q = search.value.toLowerCase()
  return data.value.groups
    .map((g: any) => ({
      category: g.category,
      apps: g.apps.filter((a: any) => !q || a.name.toLowerCase().includes(q) || a.description.toLowerCase().includes(q)),
    }))
    .filter((g: any) => g.apps.length)
})

async function load() {
  const { data: res } = await http.get('/appstore')
  data.value = res
}

async function install(app: any) {
  await ElMessageBox.confirm(
    `„${app.name}" jetzt per Docker installieren? Das Image wird gezogen und auf Port ${app.port} gestartet.`,
    'App installieren',
    { type: 'info' },
  )
  installing.value = app.key
  try {
    await http.post('/appstore/install', { key: app.key })
    ElMessage.success(`${app.name} installiert`)
    await load()
  } finally {
    installing.value = ''
  }
}

function openApp(app: any) {
  window.open(`${location.protocol}//${location.hostname}:${app.port}`, '_blank')
}

onMounted(load)
</script>

<style scoped>
.cat-title {
  margin: 0 0 12px;
  font-size: 16px;
  color: #303133;
  border-left: 3px solid #409eff;
  padding-left: 8px;
}
.app-card {
  height: 100%;
}
.app-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}
.app-desc {
  color: #909399;
  font-size: 13px;
  min-height: 52px;
  margin: 0 0 10px;
}
.app-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.port {
  color: #c0c4cc;
  font-size: 12px;
}
</style>
