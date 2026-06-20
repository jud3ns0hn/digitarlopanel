<template>
  <div>
    <h2 class="page-title">Software & Dienste</h2>
    <el-alert v-if="info" :closable="false" type="info" style="margin-bottom: 16px">
      Distribution: <b>{{ info.family }}</b> · Paketmanager: <b>{{ info.tool }}</b>
    </el-alert>

    <el-row :gutter="16">
      <el-col v-for="app in apps" :key="app.key" :span="8" style="margin-bottom: 16px">
        <el-card shadow="never">
          <template #header>
            <div style="display: flex; align-items: center; justify-content: space-between">
              <b>{{ app.name }}</b>
              <el-tag v-if="app.installed" :type="app.active ? 'success' : 'info'" size="small">
                {{ app.installed ? (app.active ? 'läuft' : 'gestoppt') : '' }}
              </el-tag>
              <el-tag v-else type="warning" size="small">nicht installiert</el-tag>
            </div>
          </template>
          <p style="color: #909399; min-height: 40px">{{ app.description }}</p>
          <div class="toolbar" style="margin: 0">
            <el-button v-if="!app.installed" type="primary" :loading="busy === app.key" @click="install(app)">
              Installieren
            </el-button>
            <template v-else>
              <el-button v-if="!app.active" type="success" :loading="busy === app.key" @click="action(app, 'start')">Start</el-button>
              <el-button v-else :loading="busy === app.key" @click="action(app, 'stop')">Stop</el-button>
              <el-button :loading="busy === app.key" @click="action(app, 'restart')">Neustart</el-button>
              <el-button type="danger" :loading="busy === app.key" @click="uninstall(app)">Entfernen</el-button>
            </template>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/client'

interface App {
  key: string
  name: string
  description: string
  installed: boolean
  active: boolean
  enabled: boolean
}

const apps = ref<App[]>([])
const info = ref<{ family: string; tool: string } | null>(null)
const busy = ref('')

async function load() {
  const { data } = await http.get('/software/list')
  apps.value = data.apps
  info.value = { family: data.family, tool: data.tool }
}

async function install(app: App) {
  busy.value = app.key
  try {
    await http.post('/software/install', { key: app.key })
    ElMessage.success(`${app.name} installiert`)
    await load()
  } finally {
    busy.value = ''
  }
}

async function uninstall(app: App) {
  await ElMessageBox.confirm(`${app.name} wirklich entfernen?`, 'Bestätigen', { type: 'warning' })
  busy.value = app.key
  try {
    await http.post('/software/uninstall', { key: app.key })
    ElMessage.success(`${app.name} entfernt`)
    await load()
  } finally {
    busy.value = ''
  }
}

async function action(app: App, act: string) {
  busy.value = app.key
  try {
    await http.post('/software/service', { key: app.key, action: act })
    ElMessage.success(`${app.name}: ${act}`)
    await load()
  } finally {
    busy.value = ''
  }
}

onMounted(load)
</script>
