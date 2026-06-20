<template>
  <div>
    <h2 class="page-title">Backups</h2>
    <div class="toolbar">
      <el-button type="primary" @click="dialog.visible = true">Neues Backup</el-button>
      <el-button @click="load">Aktualisieren</el-button>
    </div>

    <el-table :data="backups" v-loading="loading" size="small">
      <el-table-column prop="name" label="Name" />
      <el-table-column prop="type" label="Typ" width="120" />
      <el-table-column prop="source" label="Quelle" />
      <el-table-column label="Größe" width="120">
        <template #default="{ row }">{{ fmt(row.size) }}</template>
      </el-table-column>
      <el-table-column label="Erstellt" width="180">
        <template #default="{ row }">{{ new Date(row.created_at).toLocaleString() }}</template>
      </el-table-column>
      <el-table-column label="Aktionen" width="200">
        <template #default="{ row }">
          <el-button link @click="download(row)">Download</el-button>
          <el-button link type="danger" @click="remove(row)">Löschen</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-card shadow="never" style="margin-top: 24px">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>Geplante Backups</span>
          <el-button type="primary" link @click="schedDialog.visible = true">Zeitplan hinzufügen</el-button>
        </div>
      </template>
      <el-table :data="schedules" size="small">
        <el-table-column prop="name" label="Name" />
        <el-table-column prop="type" label="Typ" width="100" />
        <el-table-column prop="source" label="Quelle" />
        <el-table-column prop="schedule" label="Cron" width="140" />
        <el-table-column prop="retention" label="Behalten" width="90" />
        <el-table-column label="Letzter Lauf" width="180">
          <template #default="{ row }">
            <span v-if="row.last_run_at">{{ new Date(row.last_run_at).toLocaleString() }}</span>
            <span v-else>—</span>
            <el-tag v-if="row.last_status" :type="row.last_status === 'ok' ? 'success' : 'danger'" size="small" style="margin-left: 6px">
              {{ row.last_status === 'ok' ? 'ok' : 'Fehler' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Status" width="90">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? 'aktiv' : 'aus' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Aktionen" width="200">
          <template #default="{ row }">
            <el-button link @click="toggleSchedule(row)">{{ row.enabled ? 'Pausieren' : 'Aktivieren' }}</el-button>
            <el-button link type="danger" @click="removeSchedule(row)">Löschen</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="schedDialog.visible" title="Geplantes Backup" width="480px">
      <el-form label-width="120px">
        <el-form-item label="Name"><el-input v-model="schedDialog.name" placeholder="nightly" /></el-form-item>
        <el-form-item label="Typ">
          <el-select v-model="schedDialog.type">
            <el-option label="Dateien" value="files" />
            <el-option label="Datenbank" value="database" />
          </el-select>
        </el-form-item>
        <el-form-item :label="schedDialog.type === 'files' ? 'Pfad' : 'DB-Name'">
          <el-input v-model="schedDialog.source" :placeholder="schedDialog.type === 'files' ? '/var/www/site' : 'meine_db'" />
        </el-form-item>
        <el-form-item label="Cron">
          <el-input v-model="schedDialog.schedule" placeholder="0 3 * * *" />
        </el-form-item>
        <el-form-item label="Behalten (N)">
          <el-input-number v-model="schedDialog.retention" :min="1" :max="365" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="schedDialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" @click="createSchedule">Erstellen</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="dialog.visible" title="Neues Backup" width="480px">
      <el-form label-width="120px">
        <el-form-item label="Name"><el-input v-model="dialog.name" placeholder="backup1" /></el-form-item>
        <el-form-item label="Typ">
          <el-select v-model="dialog.type">
            <el-option label="Dateien" value="files" />
            <el-option label="Datenbank" value="database" />
          </el-select>
        </el-form-item>
        <el-form-item :label="dialog.type === 'files' ? 'Pfad' : 'DB-Name'">
          <el-input v-model="dialog.source" :placeholder="dialog.type === 'files' ? '/var/www/site' : 'meine_db'" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" :loading="saving" @click="create">Erstellen</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http, { getToken } from '../api/client'
import { formatBytes } from '../utils/format'

const backups = ref<any[]>([])
const schedules = ref<any[]>([])
const loading = ref(false)
const saving = ref(false)
const fmt = formatBytes
const dialog = reactive({ visible: false, name: '', type: 'files', source: '' })
const schedDialog = reactive({ visible: false, name: '', type: 'files', source: '', schedule: '0 3 * * *', retention: 7 })

async function load() {
  loading.value = true
  try {
    const [b, s] = await Promise.all([http.get('/backups'), http.get('/schedules')])
    backups.value = b.data
    schedules.value = s.data
  } finally {
    loading.value = false
  }
}

async function createSchedule() {
  if (!schedDialog.name || !schedDialog.source || !schedDialog.schedule) {
    ElMessage.warning('Name, Quelle und Cron erforderlich')
    return
  }
  await http.post('/schedules', {
    name: schedDialog.name,
    type: schedDialog.type,
    source: schedDialog.source,
    schedule: schedDialog.schedule,
    retention: schedDialog.retention,
  })
  ElMessage.success('Zeitplan erstellt')
  schedDialog.visible = false
  schedDialog.name = ''
  schedDialog.source = ''
  await load()
}

async function toggleSchedule(row: any) {
  await http.post(`/schedules/${row.id}/toggle`)
  await load()
}

async function removeSchedule(row: any) {
  await ElMessageBox.confirm(`Zeitplan "${row.name}" löschen?`, 'Bestätigen', { type: 'warning' })
  await http.delete(`/schedules/${row.id}`)
  ElMessage.success('Gelöscht')
  await load()
}

async function create() {
  if (!dialog.name || !dialog.source) {
    ElMessage.warning('Name und Quelle erforderlich')
    return
  }
  saving.value = true
  try {
    await http.post('/backups', { name: dialog.name, type: dialog.type, source: dialog.source })
    ElMessage.success('Backup erstellt')
    dialog.visible = false
    dialog.name = ''
    dialog.source = ''
    await load()
  } finally {
    saving.value = false
  }
}

function download(row: any) {
  window.open(`/api/backups/${row.id}/download?token=${getToken()}`, '_blank')
}

async function remove(row: any) {
  await ElMessageBox.confirm(`Backup "${row.name}" löschen?`, 'Bestätigen', { type: 'warning' })
  await http.delete(`/backups/${row.id}`)
  ElMessage.success('Gelöscht')
  await load()
}

onMounted(load)
</script>
