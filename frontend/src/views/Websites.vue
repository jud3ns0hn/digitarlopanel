<template>
  <div>
    <h2 class="page-title">Websites</h2>
    <div class="toolbar">
      <el-button type="primary" @click="dialog.visible = true">Neue Website</el-button>
      <el-button @click="load">Aktualisieren</el-button>
      <el-tag v-if="phpVersions.length" type="info">
        PHP installiert: {{ phpVersions.map((v) => v.version).join(', ') }}
      </el-tag>
      <el-tag v-else type="warning">Kein PHP-FPM erkannt</el-tag>
    </div>

    <el-table :data="sites" v-loading="loading" size="small">
      <el-table-column prop="domain" label="Domain" />
      <el-table-column prop="root" label="Document-Root" />
      <el-table-column label="PHP" width="120">
        <template #default="{ row }">
          <el-tag v-if="row.php_version" size="small">PHP {{ row.php_version }}</el-tag>
          <span v-else>—</span>
        </template>
      </el-table-column>
      <el-table-column label="Status" width="110">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? 'aktiv' : 'inaktiv' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Aktionen" width="320">
        <template #default="{ row }">
          <el-button link @click="setPHP(row)">PHP</el-button>
          <el-button link @click="toggle(row)">{{ row.enabled ? 'Deaktivieren' : 'Aktivieren' }}</el-button>
          <el-button link type="danger" @click="remove(row)">Löschen</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialog.visible" title="Neue Website" width="480px">
      <el-form label-width="120px">
        <el-form-item label="Domain"><el-input v-model="dialog.domain" placeholder="example.com" /></el-form-item>
        <el-form-item label="Document-Root">
          <el-input v-model="dialog.root" placeholder="/var/www/example.com" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" :loading="saving" @click="create">Erstellen</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="phpDialog.visible" title="PHP-Version zuweisen" width="420px">
      <el-form label-width="120px">
        <el-form-item label="Version">
          <el-select v-model="phpDialog.version" style="width: 100%">
            <el-option label="Kein PHP (deaktivieren)" value="" />
            <el-option v-for="v in phpVersions" :key="v.version" :label="`PHP ${v.version}`" :value="v.version" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="phpDialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" @click="savePHP">Speichern</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/client'

const sites = ref<any[]>([])
const phpVersions = ref<any[]>([])
const loading = ref(false)
const saving = ref(false)
const dialog = reactive({ visible: false, domain: '', root: '' })
const phpDialog = reactive({ visible: false, id: 0, version: '' })

async function load() {
  loading.value = true
  try {
    const [sitesRes, phpRes] = await Promise.all([http.get('/websites'), http.get('/php')])
    sites.value = sitesRes.data
    phpVersions.value = phpRes.data.versions || []
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
    await http.post('/websites', { domain: dialog.domain, root: dialog.root })
    ElMessage.success('Website erstellt')
    dialog.visible = false
    dialog.domain = ''
    dialog.root = ''
    await load()
  } finally {
    saving.value = false
  }
}

function setPHP(row: any) {
  phpDialog.id = row.id
  phpDialog.version = row.php_version || ''
  phpDialog.visible = true
}

async function savePHP() {
  await http.post(`/websites/${phpDialog.id}/php`, { version: phpDialog.version })
  ElMessage.success('PHP-Version aktualisiert')
  phpDialog.visible = false
  await load()
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
