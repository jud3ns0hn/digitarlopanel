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
const loading = ref(false)
const saving = ref(false)
const fmt = formatBytes
const dialog = reactive({ visible: false, name: '', type: 'files', source: '' })

async function load() {
  loading.value = true
  try {
    const { data } = await http.get('/backups')
    backups.value = data
  } finally {
    loading.value = false
  }
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
