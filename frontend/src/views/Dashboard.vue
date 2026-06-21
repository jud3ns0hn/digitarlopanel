<template>
  <div>
    <h2 class="page-title">Dashboard</h2>

    <el-row :gutter="16" v-if="host">
      <el-col :span="24">
        <el-card shadow="never" style="margin-bottom: 16px">
          <el-descriptions :column="4" size="small" border>
            <el-descriptions-item label="Hostname">{{ host.host.hostname }}</el-descriptions-item>
            <el-descriptions-item label="OS">{{ host.os.name || host.host.platform }}</el-descriptions-item>
            <el-descriptions-item label="Kernel">{{ host.host.kernel }}</el-descriptions-item>
            <el-descriptions-item label="Arch">{{ host.host.arch }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16">
      <el-col :span="6"><el-card shadow="never" class="metric-card"><div class="label">CPU</div><div class="value">{{ metrics?.cpu_percent?.toFixed(1) ?? '-' }}%</div></el-card></el-col>
      <el-col :span="6"><el-card shadow="never" class="metric-card"><div class="label">RAM</div><div class="value">{{ metrics?.memory?.used_percent?.toFixed(1) ?? '-' }}%</div></el-card></el-col>
      <el-col :span="6"><el-card shadow="never" class="metric-card"><div class="label">Load (1m)</div><div class="value">{{ metrics?.load1?.toFixed(2) ?? '-' }}</div></el-card></el-col>
      <el-col :span="6"><el-card shadow="never" class="metric-card"><div class="label">Uptime</div><div class="value">{{ uptime }}</div></el-card></el-col>
    </el-row>

    <el-row :gutter="16" style="margin-top: 16px">
      <el-col :span="12"><el-card shadow="never"><div ref="cpuChart" style="height: 280px"></div></el-card></el-col>
      <el-col :span="12"><el-card shadow="never"><div ref="memChart" style="height: 280px"></div></el-card></el-col>
    </el-row>

    <el-card shadow="never" style="margin-top: 16px">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>Verlauf</span>
          <el-radio-group v-model="historyWindow" size="small" @change="loadHistory">
            <el-radio-button :value="3600">1h</el-radio-button>
            <el-radio-button :value="21600">6h</el-radio-button>
            <el-radio-button :value="86400">24h</el-radio-button>
            <el-radio-button :value="604800">7d</el-radio-button>
          </el-radio-group>
        </div>
      </template>
      <div ref="histChart" style="height: 300px"></div>
    </el-card>

    <el-card shadow="never" style="margin-top: 16px">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>Top-Prozesse (nach Speicher)</span>
          <el-button link @click="loadProcesses">Aktualisieren</el-button>
        </div>
      </template>
      <el-table :data="processes" size="small" max-height="320">
        <el-table-column prop="pid" label="PID" width="100" />
        <el-table-column prop="name" label="Name" />
        <el-table-column prop="user" label="Benutzer" width="140" />
        <el-table-column label="CPU %" width="120">
          <template #default="{ row }">{{ row.cpu.toFixed(1) }}</template>
        </el-table-column>
        <el-table-column label="Speicher %" width="120">
          <template #default="{ row }">{{ row.memory.toFixed(1) }}</template>
        </el-table-column>
        <el-table-column v-if="auth.isAdmin" label="" width="120" align="right">
          <template #default="{ row }">
            <el-button link type="danger" size="small" @click="killProcess(row)">Beenden</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never" style="margin-top: 16px">
      <template #header>Festplatten</template>
      <el-table :data="metrics?.disks || []" size="small">
        <el-table-column prop="mount" label="Mountpoint" />
        <el-table-column label="Größe"><template #default="{ row }">{{ fmt(row.total) }}</template></el-table-column>
        <el-table-column label="Belegt"><template #default="{ row }">{{ fmt(row.used) }}</template></el-table-column>
        <el-table-column label="Auslastung" width="240">
          <template #default="{ row }">
            <el-progress :percentage="Math.round(row.used_percent)" :status="row.used_percent > 90 ? 'exception' : undefined" />
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, nextTick } from 'vue'
import * as echarts from 'echarts'
import { ElMessage, ElMessageBox } from 'element-plus'
import http, { getToken } from '../api/client'
import { formatBytes, formatUptime } from '../utils/format'
import { useAuthStore } from '../store/auth'

const auth = useAuthStore()

interface Metrics {
  cpu_percent: number
  load1: number
  memory: { used_percent: number; total: number; used: number }
  disks: { mount: string; total: number; used: number; used_percent: number }[]
  uptime_secs: number
}

const metrics = ref<Metrics | null>(null)
const host = ref<any>(null)
const processes = ref<any[]>([])
const cpuChart = ref<HTMLElement>()
const memChart = ref<HTMLElement>()
const histChart = ref<HTMLElement>()
const historyWindow = ref(3600)
const fmt = formatBytes

let cpuInstance: echarts.ECharts | null = null
let memInstance: echarts.ECharts | null = null
let histInstance: echarts.ECharts | null = null
let ws: WebSocket | null = null
const cpuSeries: number[] = []
const memSeries: number[] = []
const labels: string[] = []
const MAX_POINTS = 30

const uptime = computed(() => (metrics.value ? formatUptime(metrics.value.uptime_secs) : '-'))

function baseLineOption(title: string, color: string) {
  return {
    title: { text: title, textStyle: { fontSize: 14 } },
    grid: { left: 40, right: 16, top: 40, bottom: 30 },
    xAxis: { type: 'category', data: labels, axisLabel: { show: false } },
    yAxis: { type: 'value', max: 100, axisLabel: { formatter: '{value}%' } },
    series: [{ type: 'line', smooth: true, showSymbol: false, areaStyle: { opacity: 0.2 }, itemStyle: { color }, data: [] as number[] }],
  }
}

function pushMetric(m: Metrics) {
  metrics.value = m
  const label = new Date().toLocaleTimeString()
  labels.push(label)
  cpuSeries.push(Number(m.cpu_percent?.toFixed(1) ?? 0))
  memSeries.push(Number(m.memory?.used_percent?.toFixed(1) ?? 0))
  if (labels.length > MAX_POINTS) {
    labels.shift()
    cpuSeries.shift()
    memSeries.shift()
  }
  cpuInstance?.setOption({ xAxis: { data: labels }, series: [{ data: cpuSeries }] })
  memInstance?.setOption({ xAxis: { data: labels }, series: [{ data: memSeries }] })
}

function connect() {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  const token = getToken()
  ws = new WebSocket(`${proto}://${location.host}/api/system/metrics/stream?token=${token}`)
  ws.onmessage = (ev) => {
    try {
      pushMetric(JSON.parse(ev.data))
    } catch {
      /* ignore malformed frame */
    }
  }
  ws.onclose = () => {
    ws = null
    setTimeout(() => {
      if (!ws) connect()
    }, 3000)
  }
}

async function loadProcesses() {
  const { data } = await http.get('/system/processes')
  processes.value = data
}

async function killProcess(row: any) {
  try {
    await ElMessageBox.confirm(
      `Prozess „${row.name}" (PID ${row.pid}) beenden?`,
      'Prozess beenden',
      { type: 'warning', confirmButtonText: 'Beenden (TERM)', distinguishCancelAndClose: true },
    )
  } catch {
    return
  }
  try {
    await http.post('/system/processes/kill', { pid: row.pid, signal: 'TERM' })
    ElMessage.success('Signal gesendet')
    setTimeout(loadProcesses, 800)
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || 'Konnte Prozess nicht beenden')
  }
}

async function loadHistory() {
  const { data } = await http.get('/system/history', { params: { window: historyWindow.value } })
  const times = data.map((s: any) => new Date(s.timestamp * 1000).toLocaleString())
  histInstance?.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['CPU %', 'RAM %', 'Disk %'] },
    grid: { left: 45, right: 16, top: 40, bottom: 30 },
    xAxis: { type: 'category', data: times, axisLabel: { show: false } },
    yAxis: { type: 'value', max: 100, axisLabel: { formatter: '{value}%' } },
    series: [
      { name: 'CPU %', type: 'line', smooth: true, showSymbol: false, data: data.map((s: any) => s.cpu_percent?.toFixed(1)) },
      { name: 'RAM %', type: 'line', smooth: true, showSymbol: false, data: data.map((s: any) => s.mem_percent?.toFixed(1)) },
      { name: 'Disk %', type: 'line', smooth: true, showSymbol: false, data: data.map((s: any) => s.disk_percent?.toFixed(1)) },
    ],
  })
}

onMounted(async () => {
  const { data } = await http.get('/system/host')
  host.value = data
  loadProcesses()
  await nextTick()
  cpuInstance = echarts.init(cpuChart.value!)
  memInstance = echarts.init(memChart.value!)
  histInstance = echarts.init(histChart.value!)
  cpuInstance.setOption(baseLineOption('CPU-Auslastung', '#409eff'))
  memInstance.setOption(baseLineOption('Speicher-Auslastung', '#67c23a'))
  loadHistory()
  connect()
  window.addEventListener('resize', onResize)
})

function onResize() {
  cpuInstance?.resize()
  memInstance?.resize()
  histInstance?.resize()
}

onBeforeUnmount(() => {
  ws?.close()
  ws = null
  window.removeEventListener('resize', onResize)
  cpuInstance?.dispose()
  memInstance?.dispose()
  histInstance?.dispose()
})
</script>
