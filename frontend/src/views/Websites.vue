<template>
  <div>
    <h2 class="page-title">Websites</h2>
    <div class="toolbar">
      <el-button type="primary" @click="dialog.visible = true">Neue Website</el-button>
      <el-button @click="load">Aktualisieren</el-button>
    </div>

    <el-table :data="sites" v-loading="loading" size="small">
      <el-table-column prop="domain" label="Domain" />
      <el-table-column prop="root" label="Document-Root" />
      <el-table-column label="Status" width="120">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? 'aktiv' : 'inaktiv' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Aktionen" width="240">
        <template #default="{ row }">
          <el-button link @click="toggle(row)">{{ row.enabled ? 'Deaktivieren' : 'Aktivieren' }}</el-button>
          <el-button link type="danger" @click="remove(row)">Löschen</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialog.visible" title="Neue Website" width="480px">
      <el-form label-width="120px">
        <el-form-item label="Domain">
          <el-input v-model="dialog.domain" placeholder="example.com" />
        </el-form-item>
        <el-form-item label="Document-Root">
          <el-input v-model="dialog.root" placeholder="/var/www/example.com" />
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

const sites = ref<any[]>([])
const loading = ref(false)
const saving = ref(false)
const dialog = reactive({ visible: false, domain: '', root: '' })

async function load() {
  loading.value = true
  try {
    const { data } = await http.get('/websites')
    sites.value = data
  } finally {
    loading.value = false
  }
}

async function create() {
  if (!dialog.domain || !dialog.root) {
    ElMessage.warning('Domain und Root erforderlich')
    return
  }
  saving.value = true
  try {
    const { data } = await http.post('/websites', { domain: dialog.domain, root: dialog.root })
    if (data.reload_ok === false) {
      ElMessage.warning('Website angelegt, aber nginx-Reload fehlgeschlagen (läuft nginx?)')
    } else {
      ElMessage.success('Website erstellt')
    }
    dialog.visible = false
    dialog.domain = ''
    dialog.root = ''
    await load()
  } finally {
    saving.value = false
  }
}

async function toggle(row: any) {
  await http.post(`/websites/${row.id}/toggle`)
  await load()
}

async function remove(row: any) {
  await ElMessageBox.confirm(`Website "${row.domain}" löschen?`, 'Bestätigen', { type: 'warning' })
  await http.delete(`/websites/${row.id}`)
  ElMessage.success('Gelöscht')
  await load()
}

onMounted(load)
</script>
