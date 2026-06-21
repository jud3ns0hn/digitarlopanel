<template>
  <div>
    <h2 class="page-title">Docker</h2>

    <el-card v-if="data && !data.available" shadow="never" class="empty-card">
      <el-empty description="Docker ist auf diesem Server nicht installiert">
        <p style="color: #909399; max-width: 460px; margin: 0 auto 16px">
          Docker wird für den App-Store und App-Stacks benötigt. Ein Klick installiert
          die offizielle Docker Engine (inkl. Compose) und aktiviert den Dienst.
        </p>
        <el-button type="primary" :loading="installing" @click="installDocker">
          Docker jetzt installieren
        </el-button>
      </el-empty>
    </el-card>

    <template v-else>
      <div class="toolbar">
        <el-input v-model="pullImage" placeholder="Image ziehen, z.B. nginx:latest" style="width: 320px" />
        <el-button type="primary" :loading="pulling" @click="pull">Pull</el-button>
        <el-button @click="load">Aktualisieren</el-button>
        <el-button type="warning" @click="prune">Aufräumen (Prune)</el-button>
      </div>

      <el-card shadow="never" style="margin-bottom: 16px">
        <template #header>Container</template>
        <el-table :data="data?.containers || []" size="small">
          <el-table-column prop="name" label="Name" />
          <el-table-column prop="image" label="Image" />
          <el-table-column label="Status" width="130">
            <template #default="{ row }">
              <el-tag :type="row.state === 'running' ? 'success' : 'info'">{{ row.state }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="Detail" />
          <el-table-column prop="ports" label="Ports" />
          <el-table-column label="Aktionen" width="360">
            <template #default="{ row }">
              <el-button link type="success" @click="action(row, 'start')">Start</el-button>
              <el-button link @click="action(row, 'stop')">Stop</el-button>
              <el-button link @click="action(row, 'restart')">Neustart</el-button>
              <el-button link @click="showLogs(row)">Logs</el-button>
              <el-button link type="danger" @click="remove(row)">Entfernen</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>

      <el-row :gutter="16">
        <el-col :span="12">
          <el-card shadow="never">
            <template #header>Netzwerke</template>
            <el-table :data="networks" size="small">
              <el-table-column prop="name" label="Name" />
              <el-table-column prop="driver" label="Driver" width="110" />
              <el-table-column prop="scope" label="Scope" width="100" />
            </el-table>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card shadow="never">
            <template #header>Volumes</template>
            <el-table :data="volumes" size="small">
              <el-table-column prop="name" label="Name" />
              <el-table-column prop="driver" label="Driver" width="110" />
            </el-table>
          </el-card>
        </el-col>
      </el-row>

      <el-card shadow="never" style="margin-top: 16px">
        <template #header>Images</template>
        <el-table :data="data?.images || []" size="small">
          <el-table-column prop="repository" label="Repository" />
          <el-table-column prop="tag" label="Tag" width="160" />
          <el-table-column prop="size" label="Größe" width="120" />
        </el-table>
      </el-card>
    </template>

    <el-dialog v-model="logsDialog.visible" :title="`Logs: ${logsDialog.name}`" width="70%">
      <pre style="max-height: 480px; overflow: auto; font-size: 12px">{{ logsDialog.text }}</pre>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/client'

const data = ref<any>(null)
const networks = ref<any[]>([])
const volumes = ref<any[]>([])
const pullImage = ref('')
const pulling = ref(false)
const installing = ref(false)
const logsDialog = reactive({ visible: false, name: '', text: '' })

async function installDocker() {
  installing.value = true
  try {
    await ElMessageBox.confirm(
      'Die offizielle Docker Engine wird installiert (get.docker.com). Das kann ein bis zwei Minuten dauern.',
      'Docker installieren',
      { type: 'info', confirmButtonText: 'Installieren' },
    )
    ElMessage.info('Docker wird installiert …')
    const { data: res } = await http.post('/docker/install')
    if (res.available) {
      ElMessage.success('Docker installiert')
      await load()
    } else {
      ElMessage.warning('Installation lief, Docker aber nicht erkannt — Seite neu laden')
    }
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e?.response?.data?.error || 'Installation fehlgeschlagen')
  } finally {
    installing.value = false
  }
}

async function load() {
  const res = await http.get('/docker')
  data.value = res.data
  if (res.data.available) {
    const [n, v] = await Promise.all([http.get('/docker/networks'), http.get('/docker/volumes')])
    networks.value = n.data
    volumes.value = v.data
  }
}

async function showLogs(row: any) {
  const { data: res } = await http.get('/docker/logs', { params: { id: row.id } })
  logsDialog.name = row.name
  logsDialog.text = res.logs || '(keine Logs)'
  logsDialog.visible = true
}

async function prune() {
  await ElMessageBox.confirm('Ungenutzte Container, Netzwerke, Images und Build-Cache entfernen?', 'Aufräumen', { type: 'warning' })
  const { data: res } = await http.post('/docker/prune')
  ElMessage.success('Aufgeräumt')
  console.log(res.output)
  await load()
}

async function action(row: any, act: string) {
  await http.post('/docker/container', { id: row.id, action: act })
  ElMessage.success(`${row.name}: ${act}`)
  await load()
}

async function remove(row: any) {
  await ElMessageBox.confirm(`Container "${row.name}" entfernen?`, 'Bestätigen', { type: 'warning' })
  await http.post('/docker/container', { id: row.id, action: 'remove' })
  ElMessage.success('Entfernt')
  await load()
}

async function pull() {
  if (!pullImage.value) return
  pulling.value = true
  try {
    await http.post('/docker/pull', { image: pullImage.value })
    ElMessage.success('Image geladen')
    pullImage.value = ''
    await load()
  } finally {
    pulling.value = false
  }
}

onMounted(load)
</script>
