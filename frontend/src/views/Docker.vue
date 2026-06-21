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
          <el-table-column label="Aktionen" width="400" align="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="openDetail(row)">Details</el-button>
              <el-button link type="success" @click="action(row, 'start')">Start</el-button>
              <el-button link @click="action(row, 'stop')">Stop</el-button>
              <el-button link @click="action(row, 'restart')">Neustart</el-button>
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

    <el-drawer v-model="detailDrawer.visible" :title="detailDrawer.name" size="640px" @open="loadInspect">
      <el-tabs v-model="detailDrawer.tab">
        <el-tab-pane label="Details" name="details">
          <el-skeleton v-if="!inspect" :rows="6" animated />
          <template v-else>
            <el-descriptions :column="1" border size="small">
              <el-descriptions-item label="Name">{{ inspect.name }}</el-descriptions-item>
              <el-descriptions-item label="Image">{{ inspect.image }}</el-descriptions-item>
              <el-descriptions-item label="Status">
                <el-tag :type="inspect.state === 'running' ? 'success' : 'info'" size="small">{{ inspect.state }}</el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="Restart-Policy">{{ inspect.restart_policy || '–' }}</el-descriptions-item>
              <el-descriptions-item label="Erstellt">{{ formatDate(inspect.created) }}</el-descriptions-item>
              <el-descriptions-item label="Command">{{ inspect.command || '–' }}</el-descriptions-item>
              <el-descriptions-item label="Netzwerke">{{ (inspect.networks || []).join(', ') || '–' }}</el-descriptions-item>
            </el-descriptions>

            <el-divider content-position="left">Ports</el-divider>
            <el-empty v-if="!portList.length" description="Keine veröffentlichten Ports" :image-size="50" />
            <el-tag v-for="p in portList" :key="p" style="margin: 2px">{{ p }}</el-tag>

            <el-divider content-position="left">Volumes / Mounts</el-divider>
            <el-empty v-if="!(inspect.mounts || []).length" description="Keine Mounts" :image-size="50" />
            <div v-for="m in inspect.mounts" :key="m" class="mono-line">{{ m }}</div>

            <el-divider content-position="left">Umgebungsvariablen</el-divider>
            <div v-for="e in inspect.env" :key="e" class="mono-line">{{ e }}</div>

            <el-divider content-position="left">Live-Stats</el-divider>
            <pre class="statbox">{{ stats || 'lade …' }}</pre>
          </template>
        </el-tab-pane>
        <el-tab-pane label="Logs" name="logs">
          <el-button :icon="undefined" size="small" @click="loadDrawerLogs" style="margin-bottom: 8px">Aktualisieren</el-button>
          <pre class="logbox">{{ drawerLogs || '(keine Logs)' }}</pre>
        </el-tab-pane>
      </el-tabs>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/client'

const data = ref<any>(null)
const networks = ref<any[]>([])
const volumes = ref<any[]>([])
const pullImage = ref('')
const pulling = ref(false)
const installing = ref(false)

const detailDrawer = reactive<{ visible: boolean; tab: string; id: string; name: string }>({
  visible: false, tab: 'details', id: '', name: '',
})
const inspect = ref<any>(null)
const stats = ref('')
const drawerLogs = ref('')

const portList = computed(() => {
  if (!inspect.value?.ports) return []
  return Object.entries(inspect.value.ports).map(([k, v]) => (v ? `${v} → ${k}` : k))
})

function formatDate(s?: string) {
  if (!s) return '–'
  const d = new Date(s)
  return isNaN(d.getTime()) ? s : d.toLocaleString()
}

function openDetail(row: any) {
  detailDrawer.id = row.id
  detailDrawer.name = row.name
  detailDrawer.tab = 'details'
  inspect.value = null
  stats.value = ''
  drawerLogs.value = ''
  detailDrawer.visible = true
}

async function loadInspect() {
  const { data } = await http.get('/docker/inspect', { params: { id: detailDrawer.id } })
  inspect.value = data.detail
  stats.value = data.stats || '(keine Stats)'
  await loadDrawerLogs()
}

async function loadDrawerLogs() {
  const { data } = await http.get('/docker/logs', { params: { id: detailDrawer.id } })
  drawerLogs.value = data.logs || ''
}

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

<style scoped>
.empty-card {
  padding: 32px 0;
}
.mono-line {
  font-family: ui-monospace, monospace;
  font-size: 12px;
  padding: 2px 0;
  word-break: break-all;
  color: #606266;
}
.logbox,
.statbox {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 12px;
  border-radius: 6px;
  font-size: 12px;
  line-height: 1.5;
  max-height: 420px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
}
.statbox {
  max-height: 160px;
}
</style>
