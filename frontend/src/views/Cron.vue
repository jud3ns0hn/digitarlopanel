<template>
  <div>
    <h2 class="page-title">Cron-Jobs</h2>
    <div class="toolbar">
      <el-button type="primary" @click="dialog.visible = true">Neuer Job</el-button>
      <el-button @click="load">Aktualisieren</el-button>
    </div>

    <el-table :data="jobs" v-loading="loading" size="small">
      <el-table-column prop="name" label="Name" width="180" />
      <el-table-column prop="schedule" label="Zeitplan" width="160" />
      <el-table-column prop="command" label="Befehl" />
      <el-table-column label="Status" width="100">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? 'aktiv' : 'inaktiv' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Aktionen" width="220">
        <template #default="{ row }">
          <el-button link @click="toggle(row)">{{ row.enabled ? 'Deaktivieren' : 'Aktivieren' }}</el-button>
          <el-button link type="danger" @click="remove(row)">Löschen</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialog.visible" title="Neuer Cron-Job" width="520px">
      <el-form label-width="120px">
        <el-form-item label="Name"><el-input v-model="dialog.name" placeholder="Backup" /></el-form-item>
        <el-form-item label="Zeitplan">
          <el-input v-model="dialog.schedule" placeholder="0 3 * * *" />
        </el-form-item>
        <el-form-item label="Befehl">
          <el-input v-model="dialog.command" type="textarea" placeholder="/usr/local/bin/backup.sh" />
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
import http from '../api/client'

const jobs = ref<any[]>([])
const loading = ref(false)
const saving = ref(false)
const dialog = reactive({ visible: false, name: '', schedule: '', command: '' })

async function load() {
  loading.value = true
  try {
    const { data } = await http.get('/cron')
    jobs.value = data
  } finally {
    loading.value = false
  }
}

async function create() {
  if (!dialog.name || !dialog.schedule || !dialog.command) {
    ElMessage.warning('Alle Felder erforderlich')
    return
  }
  saving.value = true
  try {
    await http.post('/cron', { name: dialog.name, schedule: dialog.schedule, command: dialog.command })
    ElMessage.success('Job erstellt')
    dialog.visible = false
    dialog.name = ''
    dialog.schedule = ''
    dialog.command = ''
    await load()
  } finally {
    saving.value = false
  }
}

async function toggle(row: any) {
  await http.post(`/cron/${row.id}/toggle`)
  await load()
}

async function remove(row: any) {
  await ElMessageBox.confirm(`Job "${row.name}" löschen?`, 'Bestätigen', { type: 'warning' })
  await http.delete(`/cron/${row.id}`)
  ElMessage.success('Gelöscht')
  await load()
}

onMounted(load)
</script>
