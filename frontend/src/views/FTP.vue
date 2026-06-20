<template>
  <div>
    <h2 class="page-title">FTP-Konten</h2>
    <el-alert v-if="data && !data.available" :closable="false" type="warning" show-icon style="margin-bottom: 16px">
      pure-ftpd (pure-pw) ist nicht installiert. Über „Software" bzw. die Shell installierbar.
    </el-alert>
    <div class="toolbar">
      <el-button type="primary" :disabled="!available" @click="dialog.visible = true">Neues Konto</el-button>
      <el-button @click="load">Aktualisieren</el-button>
    </div>

    <el-table :data="data?.accounts || []" v-loading="loading" size="small">
      <el-table-column prop="username" label="Benutzer" />
      <el-table-column prop="home" label="Home-Verzeichnis" />
      <el-table-column label="Aktionen" width="220">
        <template #default="{ row }">
          <el-button link @click="setPassword(row)">Passwort</el-button>
          <el-button link type="danger" @click="remove(row)">Löschen</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialog.visible" title="Neues FTP-Konto" width="460px">
      <el-form label-width="120px">
        <el-form-item label="Benutzer"><el-input v-model="dialog.username" /></el-form-item>
        <el-form-item label="Home"><el-input v-model="dialog.home" placeholder="/var/www/site" /></el-form-item>
        <el-form-item label="Passwort"><el-input v-model="dialog.password" type="password" show-password /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" @click="create">Erstellen</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/client'

const data = ref<any>(null)
const loading = ref(false)
const dialog = reactive({ visible: false, username: '', home: '', password: '' })
const available = computed(() => data.value?.available)

async function load() {
  loading.value = true
  try {
    const res = await http.get('/ftp')
    data.value = res.data
  } finally {
    loading.value = false
  }
}

async function create() {
  if (!dialog.username || !dialog.home || !dialog.password) {
    ElMessage.warning('Alle Felder erforderlich')
    return
  }
  await http.post('/ftp', { username: dialog.username, home: dialog.home, password: dialog.password })
  ElMessage.success('Konto erstellt')
  dialog.visible = false
  dialog.username = ''
  dialog.home = ''
  dialog.password = ''
  await load()
}

async function setPassword(row: any) {
  const { value } = await ElMessageBox.prompt(`Neues Passwort für ${row.username}`, 'Passwort', { inputType: 'password' })
  await http.post(`/ftp/${row.id}/password`, { password: value })
  ElMessage.success('Passwort geändert')
}

async function remove(row: any) {
  await ElMessageBox.confirm(`Konto "${row.username}" löschen?`, 'Bestätigen', { type: 'warning' })
  await http.delete(`/ftp/${row.id}`)
  ElMessage.success('Gelöscht')
  await load()
}

onMounted(load)
</script>
