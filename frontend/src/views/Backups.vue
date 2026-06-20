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
      <el-table-column label="Aktionen" width="280">
        <template #default="{ row }">
          <el-button link @click="download(row)">Download</el-button>
          <el-button link :disabled="!destinations.length" @click="openUpload(row)">Hochladen</el-button>
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

    <el-card shadow="never" style="margin-top: 24px">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>Externe Backup-Ziele (S3 / SFTP / WebDAV)</span>
          <el-button type="primary" link @click="destDialog.visible = true">Ziel hinzufügen</el-button>
        </div>
      </template>
      <el-table :data="destinations" size="small">
        <el-table-column prop="name" label="Name" />
        <el-table-column prop="type" label="Typ" width="100" />
        <el-table-column prop="endpoint" label="Endpunkt" show-overflow-tooltip />
        <el-table-column prop="bucket" label="Bucket/Pfad" />
        <el-table-column label="Aktionen" width="200">
          <template #default="{ row }">
            <el-button link :loading="testing === row.id" @click="testDest(row)">Testen</el-button>
            <el-button link type="danger" @click="removeDest(row)">Löschen</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="destDialog.visible" title="Backup-Ziel" width="520px">
      <el-form label-width="130px">
        <el-form-item label="Name"><el-input v-model="destDialog.name" placeholder="offsite-s3" /></el-form-item>
        <el-form-item label="Typ">
          <el-select v-model="destDialog.type" style="width: 100%">
            <el-option label="S3-kompatibel" value="s3" />
            <el-option label="SFTP / SSH" value="sftp" />
            <el-option label="WebDAV" value="webdav" />
          </el-select>
        </el-form-item>
        <el-form-item label="Endpunkt">
          <el-input v-model="destDialog.endpoint" :placeholder="endpointHint" />
        </el-form-item>
        <el-form-item :label="destDialog.type === 's3' ? 'Bucket' : 'Verzeichnis'">
          <el-input v-model="destDialog.bucket" placeholder="mein-bucket bzw. /backups" />
        </el-form-item>
        <el-form-item v-if="destDialog.type === 's3'" label="Region">
          <el-input v-model="destDialog.region" placeholder="us-east-1" />
        </el-form-item>
        <el-form-item :label="destDialog.type === 's3' ? 'Access-Key' : 'Benutzer'">
          <el-input v-model="destDialog.access_key" />
        </el-form-item>
        <el-form-item :label="destDialog.type === 's3' ? 'Secret-Key' : 'Passwort'">
          <el-input v-model="destDialog.secret_key" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="destDialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" @click="createDest">Speichern</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="uploadDialog.visible" title="Backup hochladen" width="420px">
      <el-form label-width="100px">
        <el-form-item label="Ziel">
          <el-select v-model="uploadDialog.destId" style="width: 100%">
            <el-option v-for="d in destinations" :key="d.id" :label="`${d.name} (${d.type})`" :value="d.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="uploadDialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" :loading="uploading" @click="doUpload">Hochladen</el-button>
      </template>
    </el-dialog>

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
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http, { getToken } from '../api/client'
import { formatBytes } from '../utils/format'

const backups = ref<any[]>([])
const schedules = ref<any[]>([])
const destinations = ref<any[]>([])
const loading = ref(false)
const saving = ref(false)
const uploading = ref(false)
const testing = ref<number | null>(null)
const fmt = formatBytes
const dialog = reactive({ visible: false, name: '', type: 'files', source: '' })
const schedDialog = reactive({ visible: false, name: '', type: 'files', source: '', schedule: '0 3 * * *', retention: 7 })
const destDialog = reactive({
  visible: false,
  name: '',
  type: 's3',
  endpoint: '',
  bucket: '',
  region: 'us-east-1',
  access_key: '',
  secret_key: '',
})
const uploadDialog = reactive({ visible: false, backupId: 0, destId: 0 })

const endpointHint = computed(() => {
  if (destDialog.type === 's3') return 's3.eu-central-1.amazonaws.com (leer = AWS-Standard)'
  if (destDialog.type === 'sftp') return 'host:22'
  return 'https://dav.example.com/remote.php/dav'
})

async function load() {
  loading.value = true
  try {
    const [b, s, d] = await Promise.all([
      http.get('/backups'),
      http.get('/schedules'),
      http.get('/destinations'),
    ])
    backups.value = b.data
    schedules.value = s.data
    destinations.value = d.data
  } finally {
    loading.value = false
  }
}

async function createDest() {
  if (!destDialog.name) {
    ElMessage.warning('Name erforderlich')
    return
  }
  await http.post('/destinations', {
    name: destDialog.name,
    type: destDialog.type,
    endpoint: destDialog.endpoint,
    bucket: destDialog.bucket,
    region: destDialog.region,
    access_key: destDialog.access_key,
    secret_key: destDialog.secret_key,
  })
  ElMessage.success('Ziel gespeichert')
  destDialog.visible = false
  destDialog.name = ''
  destDialog.secret_key = ''
  await load()
}

async function testDest(row: any) {
  testing.value = row.id
  try {
    const { data } = await http.post(`/destinations/${row.id}/test`)
    if (data.ok) ElMessage.success('Verbindung erfolgreich')
    else ElMessage.error('Fehlgeschlagen: ' + data.error)
  } finally {
    testing.value = null
  }
}

async function removeDest(row: any) {
  await ElMessageBox.confirm(`Ziel "${row.name}" löschen?`, 'Bestätigen', { type: 'warning' })
  await http.delete(`/destinations/${row.id}`)
  ElMessage.success('Gelöscht')
  await load()
}

function openUpload(row: any) {
  uploadDialog.backupId = row.id
  uploadDialog.destId = destinations.value[0]?.id || 0
  uploadDialog.visible = true
}

async function doUpload() {
  if (!uploadDialog.destId) {
    ElMessage.warning('Ziel wählen')
    return
  }
  uploading.value = true
  try {
    await http.post(`/backups/${uploadDialog.backupId}/upload`, { destination_id: uploadDialog.destId })
    ElMessage.success('Hochgeladen')
    uploadDialog.visible = false
  } finally {
    uploading.value = false
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
