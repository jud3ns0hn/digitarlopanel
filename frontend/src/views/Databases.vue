<template>
  <div>
    <h2 class="page-title">Datenbanken</h2>
    <div class="toolbar">
      <el-button type="primary" @click="dialog.visible = true">Neue Datenbank</el-button>
      <el-button @click="load">Aktualisieren</el-button>
    </div>

    <el-table :data="dbs" v-loading="loading" size="small">
      <el-table-column prop="name" label="Name" />
      <el-table-column prop="username" label="Benutzer" />
      <el-table-column prop="charset" label="Charset" width="120" />
      <el-table-column label="Aktionen" width="220" align="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openTables(row)">Tabellen</el-button>
          <el-button link type="danger" @click="remove(row)">Löschen</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-drawer v-model="tablesDrawer.visible" :title="`Tabellen: ${tablesDrawer.name}`" size="560px" @open="loadTables">
      <el-skeleton v-if="tablesLoading" :rows="5" animated />
      <template v-else>
        <el-empty v-if="!tables.length" description="Keine Tabellen in dieser Datenbank" />
        <el-table v-else :data="tables" size="small">
          <el-table-column prop="name" label="Tabelle" />
          <el-table-column prop="rows" label="Zeilen (≈)" width="120" align="right" />
          <el-table-column label="Größe" width="120" align="right">
            <template #default="{ row }">{{ row.size_mb }} MB</template>
          </el-table-column>
        </el-table>
      </template>
    </el-drawer>

    <el-dialog v-model="dialog.visible" title="Neue Datenbank" width="480px">
      <el-form label-width="120px">
        <el-form-item label="Name"><el-input v-model="dialog.name" placeholder="meine_db" /></el-form-item>
        <el-form-item label="Benutzer"><el-input v-model="dialog.username" placeholder="meine_db_user" /></el-form-item>
        <el-form-item label="Passwort">
          <el-input v-model="dialog.password" placeholder="leer = automatisch generieren" />
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

const dbs = ref<any[]>([])
const loading = ref(false)
const saving = ref(false)
const dialog = reactive({ visible: false, name: '', username: '', password: '' })

const tablesDrawer = reactive({ visible: false, name: '' })
const tables = ref<any[]>([])
const tablesLoading = ref(false)

function openTables(row: any) {
  tablesDrawer.name = row.name
  tables.value = []
  tablesDrawer.visible = true
}

async function loadTables() {
  tablesLoading.value = true
  try {
    const { data } = await http.get('/databases/tables', { params: { name: tablesDrawer.name } })
    tables.value = data.tables || []
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || 'Konnte Tabellen nicht laden')
  } finally {
    tablesLoading.value = false
  }
}

async function load() {
  loading.value = true
  try {
    const { data } = await http.get('/databases')
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
    const { data } = await http.post('/databases', {
      name: dialog.name,
      username: dialog.username,
      password: dialog.password,
    })
    await ElMessageBox.alert(
      `Datenbank "${data.database.name}" angelegt.<br/>Benutzer: <b>${data.database.username}</b><br/>Passwort: <b>${data.password}</b>`,
      'Zugangsdaten (jetzt notieren!)',
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
  await ElMessageBox.confirm(`Datenbank "${row.name}" und Benutzer löschen?`, 'Bestätigen', { type: 'warning' })
  await http.delete(`/databases/${row.id}`)
  ElMessage.success('Gelöscht')
  await load()
}

onMounted(load)
</script>
