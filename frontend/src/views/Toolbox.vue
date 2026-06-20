<template>
  <div>
    <h2 class="page-title">System-Toolbox</h2>
    <div class="toolbar">
      <el-button @click="load">Aktualisieren</el-button>
    </div>

    <el-row :gutter="16">
      <el-col :span="12">
        <el-card shadow="never">
          <template #header>Zeitzone &amp; Hostname</template>
          <el-form label-width="120px">
            <el-form-item label="Zeitzone">
              <el-select
                v-model="timezone"
                filterable
                style="width: 100%"
                placeholder="Zeitzone wählen"
              >
                <el-option v-for="z in timezones" :key="z" :label="z" :value="z" />
              </el-select>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="saving.tz" @click="saveTimezone">Zeitzone setzen</el-button>
            </el-form-item>
            <el-divider />
            <el-form-item label="Hostname">
              <el-input v-model="hostname" placeholder="server.example.com" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="saving.host" @click="saveHostname">Hostname setzen</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card shadow="never">
          <template #header>Swap-Speicher</template>
          <el-descriptions :column="1" border size="small">
            <el-descriptions-item label="RAM gesamt">{{ fmtBytes(sys.mem_total) }}</el-descriptions-item>
            <el-descriptions-item label="Swap gesamt">{{ fmtBytes(sys.swap_total) }}</el-descriptions-item>
            <el-descriptions-item label="Swap belegt">{{ fmtBytes(sys.swap_used) }}</el-descriptions-item>
          </el-descriptions>
          <el-form label-width="120px" style="margin-top: 16px">
            <el-form-item label="Größe (MB)">
              <el-input-number v-model="swapSize" :min="64" :max="65536" :step="256" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="saving.swap" @click="createSwap">Swap erstellen</el-button>
              <el-button type="danger" plain @click="disableSwap">Swap entfernen</el-button>
            </el-form-item>
          </el-form>
        </el-card>

        <el-card shadow="never" style="margin-top: 16px">
          <template #header>SSH-Konfiguration (sshd -T)</template>
          <el-alert v-if="!ssh.available" :closable="false" type="warning" show-icon>
            SSH-Daemon nicht erkannt.
          </el-alert>
          <el-descriptions v-else :column="1" border size="small">
            <el-descriptions-item v-for="(v, k) in ssh.settings" :key="k" :label="String(k)">
              {{ v }}
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/client'

const sys = ref<any>({})
const ssh = ref<any>({ available: false, settings: {} })
const timezones = ref<string[]>([])
const timezone = ref('')
const hostname = ref('')
const swapSize = ref(2048)
const saving = reactive({ tz: false, host: false, swap: false })

function fmtBytes(n?: number) {
  if (!n) return '—'
  const u = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < u.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(1)} ${u[i]}`
}

async function load() {
  const [sysRes, tzRes, sshRes] = await Promise.all([
    http.get('/toolbox/system'),
    http.get('/toolbox/timezones'),
    http.get('/toolbox/ssh'),
  ])
  sys.value = sysRes.data
  timezone.value = sysRes.data.timezone || ''
  hostname.value = sysRes.data.hostname || ''
  timezones.value = tzRes.data.timezones || []
  ssh.value = sshRes.data
}

async function saveTimezone() {
  saving.tz = true
  try {
    await http.post('/toolbox/timezone', { timezone: timezone.value })
    ElMessage.success('Zeitzone gesetzt')
    await load()
  } finally {
    saving.tz = false
  }
}

async function saveHostname() {
  saving.host = true
  try {
    await http.post('/toolbox/hostname', { hostname: hostname.value })
    ElMessage.success('Hostname gesetzt')
    await load()
  } finally {
    saving.host = false
  }
}

async function createSwap() {
  saving.swap = true
  try {
    await http.post('/toolbox/swap', { size_mb: swapSize.value })
    ElMessage.success('Swap erstellt')
    await load()
  } finally {
    saving.swap = false
  }
}

async function disableSwap() {
  await ElMessageBox.confirm('Swap-Datei /var/swapfile entfernen?', 'Bestätigen', { type: 'warning' })
  await http.post('/toolbox/swap', { disable: true })
  ElMessage.success('Swap entfernt')
  await load()
}

onMounted(load)
</script>
