<template>
  <div>
    <h2 class="page-title">PostgreSQL</h2>
    <div class="toolbar">
      <el-button type="primary" @click="dialog.visible = true">Neue Datenbank</el-button>
      <el-button @click="load">Aktualisieren</el-button>
    </div>
    <el-table :data="dbs" v-loading="loading" size="small">
      <el-table-column prop="name" label="Name" />
      <el-table-column prop="username" label="Benutzer" />
      <el-table-column label="Aktionen" width="160">
        <template #default="{ row }">
          <el-button link type="danger" @click="remove(row)">Löschen</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialog.visible" title="Neue PostgreSQL-Datenbank" width="480px">
      <el-form label-width="120px">
        <el-form-item label="Name"><el-input v-model="dialog.name" /></el-form-item>
        <el-form-item label="Benutzer"><el-input v-model="dialog.username" /></el-form-item>
        <el-form-item label="Passwort"><el-input v-model="dialog.password" placeholder="leer = automatisch" /></el-form-item>
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

const dbs = ref<any[]>([])
const loading = ref(false)
const saving = ref(false)
const dialog = reactive({ visible: false, name: '', username: '', password: '' })

async function load() {
  loading.value = true
  try {
    const { data } = await http.get('/postgres')
    dbs.value = data
  } finally {
    loading.value = false
  }
}
async function create() {
  if (!dialog.name || !dialog.username) {
    ElMessage.warning('Name und Benutzer erforderlich')
    return
  }
  saving.value = true
  try {
    const { data } = await http.post('/postgres', { name: dialog.name, username: dialog.username, password: dialog.password })
    await ElMessageBox.alert(
      `DB „${data.database.name}" angelegt.<br/>Benutzer: <b>${data.database.username}</b><br/>Passwort: <b>${data.password}</b>`,
      'Zugangsdaten (notieren!)',
      { dangerouslyUseHTMLString: true },
    )
    dialog.visible = false
    dialog.name = ''
    dialog.username = ''
    dialog.password = ''
    await load()
  } finally {
    saving.value = false
  }
}
async function remove(row: any) {
  await ElMessageBox.confirm(`Datenbank „${row.name}" löschen?`, 'Bestätigen', { type: 'warning' })
  await http.delete(`/postgres/${row.id}`)
  ElMessage.success('Gelöscht')
  await load()
}
onMounted(load)
</script>
