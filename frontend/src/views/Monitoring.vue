<template>
  <div>
    <h2 class="page-title">Monitoring &amp; Alarme</h2>

    <el-tabs v-model="tab">
      <!-- Uptime monitors -->
      <el-tab-pane label="Uptime" name="uptime">
        <div class="toolbar">
          <el-button type="primary" @click="monDialog.visible = true">Neuer Monitor</el-button>
          <el-button @click="loadMonitors">Aktualisieren</el-button>
        </div>
        <el-table :data="monitors" size="small">
          <el-table-column prop="name" label="Name" />
          <el-table-column prop="url" label="URL" show-overflow-tooltip />
          <el-table-column label="Status" width="100">
            <template #default="{ row }">
              <el-tag v-if="row.last_status === 'up'" type="success" size="small">up</el-tag>
              <el-tag v-else-if="row.last_status === 'down'" type="danger" size="small">down</el-tag>
              <el-tag v-else type="info" size="small">—</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="Code" width="80">
            <template #default="{ row }">{{ row.last_code || '—' }}</template>
          </el-table-column>
          <el-table-column label="Antwort" width="100">
            <template #default="{ row }">{{ row.last_ms ? row.last_ms + ' ms' : '—' }}</template>
          </el-table-column>
          <el-table-column label="Intervall" width="100">
            <template #default="{ row }">{{ row.interval_s }}s</template>
          </el-table-column>
          <el-table-column label="Aktionen" width="200">
            <template #default="{ row }">
              <el-button link @click="toggleMonitor(row)">{{ row.enabled ? 'Pausieren' : 'Aktivieren' }}</el-button>
              <el-button link type="danger" @click="deleteMonitor(row)">Löschen</el-button>
            </template>
          </el-table-column>
        </el-table>

        <el-dialog v-model="monDialog.visible" title="Neuer Monitor" width="480px">
          <el-form label-width="120px">
            <el-form-item label="Name"><el-input v-model="monDialog.name" /></el-form-item>
            <el-form-item label="URL"><el-input v-model="monDialog.url" placeholder="https://example.com/health" /></el-form-item>
            <el-form-item label="Intervall (s)"><el-input-number v-model="monDialog.interval_s" :min="30" :max="3600" :step="30" /></el-form-item>
          </el-form>
          <template #footer>
            <el-button @click="monDialog.visible = false">Abbrechen</el-button>
            <el-button type="primary" @click="createMonitor">Erstellen</el-button>
          </template>
        </el-dialog>
      </el-tab-pane>

      <!-- Alert rules -->
      <el-tab-pane label="Alarme" name="alerts">
        <div class="toolbar">
          <el-button type="primary" @click="alertDialog.visible = true">Neue Regel</el-button>
          <el-button @click="loadAlerts">Aktualisieren</el-button>
        </div>
        <el-alert :closable="false" type="info" show-icon style="margin-bottom: 12px">
          E-Mail-Alarme benötigen SMTP-Einstellungen (Einstellungen → SMTP). Webhook-Alarme funktionieren ohne.
        </el-alert>
        <el-table :data="alerts" size="small">
          <el-table-column prop="metric" label="Metrik" width="120" />
          <el-table-column label="Schwelle" width="100">
            <template #default="{ row }">{{ row.threshold }}%</template>
          </el-table-column>
          <el-table-column prop="channel" label="Kanal" width="100" />
          <el-table-column prop="target" label="Ziel" show-overflow-tooltip />
          <el-table-column label="Aktiv" width="80">
            <template #default="{ row }">
              <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? 'ja' : 'nein' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="Aktionen" width="200">
            <template #default="{ row }">
              <el-button link @click="toggleAlert(row)">{{ row.enabled ? 'Pausieren' : 'Aktivieren' }}</el-button>
              <el-button link type="danger" @click="deleteAlert(row)">Löschen</el-button>
            </template>
          </el-table-column>
        </el-table>

        <el-dialog v-model="alertDialog.visible" title="Neue Alarm-Regel" width="480px">
          <el-form label-width="120px">
            <el-form-item label="Metrik">
              <el-select v-model="alertDialog.metric" style="width: 100%">
                <el-option label="CPU" value="cpu" />
                <el-option label="Arbeitsspeicher" value="memory" />
                <el-option label="Festplatte" value="disk" />
              </el-select>
            </el-form-item>
            <el-form-item label="Schwelle (%)"><el-input-number v-model="alertDialog.threshold" :min="1" :max="100" /></el-form-item>
            <el-form-item label="Kanal">
              <el-select v-model="alertDialog.channel" style="width: 100%">
                <el-option label="E-Mail" value="email" />
                <el-option label="Webhook" value="webhook" />
              </el-select>
            </el-form-item>
            <el-form-item :label="alertDialog.channel === 'email' ? 'E-Mail' : 'Webhook-URL'">
              <el-input v-model="alertDialog.target" />
            </el-form-item>
          </el-form>
          <template #footer>
            <el-button @click="alertDialog.visible = false">Abbrechen</el-button>
            <el-button type="primary" @click="createAlert">Erstellen</el-button>
          </template>
        </el-dialog>
      </el-tab-pane>

      <!-- GPU -->
      <el-tab-pane label="GPU" name="gpu">
        <div class="toolbar">
          <el-button @click="loadGPU">Aktualisieren</el-button>
        </div>
        <el-alert v-if="!gpu.available" :closable="false" type="info" show-icon>
          Keine NVIDIA-GPU erkannt (nvidia-smi nicht verfügbar).
        </el-alert>
        <el-table v-else :data="gpu.gpus" size="small">
          <el-table-column prop="name" label="GPU" />
          <el-table-column prop="utilization" label="Auslastung" width="120" />
          <el-table-column prop="memory_used" label="VRAM belegt" width="140" />
          <el-table-column prop="memory_total" label="VRAM gesamt" width="140" />
          <el-table-column prop="temperature" label="Temperatur" width="120" />
        </el-table>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/client'

const tab = ref('uptime')
const monitors = ref<any[]>([])
const alerts = ref<any[]>([])
const gpu = ref<any>({ available: false, gpus: [] })

const monDialog = reactive({ visible: false, name: '', url: '', interval_s: 60 })
const alertDialog = reactive({ visible: false, metric: 'cpu', threshold: 90, channel: 'email', target: '' })

async function loadMonitors() {
  monitors.value = (await http.get('/monitors')).data
}
async function loadAlerts() {
  alerts.value = (await http.get('/alerts')).data
}
async function loadGPU() {
  gpu.value = (await http.get('/gpu')).data
}

async function createMonitor() {
  if (!monDialog.name || !monDialog.url) {
    ElMessage.warning('Name und URL erforderlich')
    return
  }
  await http.post('/monitors', { name: monDialog.name, url: monDialog.url, interval_s: monDialog.interval_s })
  ElMessage.success('Monitor erstellt')
  monDialog.visible = false
  monDialog.name = ''
  monDialog.url = ''
  await loadMonitors()
}

async function toggleMonitor(row: any) {
  await http.post(`/monitors/${row.id}/toggle`)
  await loadMonitors()
}

async function deleteMonitor(row: any) {
  await ElMessageBox.confirm(`Monitor "${row.name}" löschen?`, 'Bestätigen', { type: 'warning' })
  await http.delete(`/monitors/${row.id}`)
  await loadMonitors()
}

async function createAlert() {
  if (!alertDialog.target) {
    ElMessage.warning('Ziel erforderlich')
    return
  }
  await http.post('/alerts', {
    metric: alertDialog.metric,
    threshold: alertDialog.threshold,
    channel: alertDialog.channel,
    target: alertDialog.target,
  })
  ElMessage.success('Regel erstellt')
  alertDialog.visible = false
  alertDialog.target = ''
  await loadAlerts()
}

async function toggleAlert(row: any) {
  await http.post(`/alerts/${row.id}/toggle`)
  await loadAlerts()
}

async function deleteAlert(row: any) {
  await ElMessageBox.confirm('Alarm-Regel löschen?', 'Bestätigen', { type: 'warning' })
  await http.delete(`/alerts/${row.id}`)
  await loadAlerts()
}

onMounted(() => {
  loadMonitors()
  loadAlerts()
  loadGPU()
})
</script>
