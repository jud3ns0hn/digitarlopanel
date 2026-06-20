<template>
  <div>
    <h2 class="page-title">Benutzer</h2>
    <div class="toolbar">
      <el-button type="primary" @click="dialog.visible = true">Neuer Benutzer</el-button>
      <el-button @click="load">Aktualisieren</el-button>
    </div>

    <el-table :data="users" v-loading="loading" size="small">
      <el-table-column prop="username" label="Benutzername" />
      <el-table-column label="Rolle" width="160">
        <template #default="{ row }">
          <el-tag :type="roleColor(row.role)">{{ row.role }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="2FA" width="80">
        <template #default="{ row }">
          <el-icon v-if="row.two_fa_enabled" color="#67c23a"><CircleCheck /></el-icon>
          <span v-else>—</span>
        </template>
      </el-table-column>
      <el-table-column label="Letzter Login" width="180">
        <template #default="{ row }">{{ row.last_login_at ? new Date(row.last_login_at).toLocaleString() : '—' }}</template>
      </el-table-column>
      <el-table-column label="Aktionen" width="320">
        <template #default="{ row }">
          <el-select :model-value="row.role" size="small" style="width: 120px" @change="(v: string) => setRole(row, v)">
            <el-option label="admin" value="admin" />
            <el-option label="operator" value="operator" />
            <el-option label="viewer" value="viewer" />
          </el-select>
          <el-button link @click="resetPassword(row)">Passwort</el-button>
          <el-button link type="danger" @click="remove(row)">Löschen</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialog.visible" title="Neuer Benutzer" width="480px">
      <el-form label-width="120px">
        <el-form-item label="Benutzername"><el-input v-model="dialog.username" /></el-form-item>
        <el-form-item label="Passwort"><el-input v-model="dialog.password" type="password" show-password /></el-form-item>
        <el-form-item label="Rolle">
          <el-select v-model="dialog.role">
            <el-option label="admin" value="admin" />
            <el-option label="operator" value="operator" />
            <el-option label="viewer" value="viewer" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" @click="create">Erstellen</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/client'

const users = ref<any[]>([])
const loading = ref(false)
const dialog = reactive({ visible: false, username: '', password: '', role: 'viewer' })

function roleColor(role: string) {
  return role === 'admin' ? 'danger' : role === 'operator' ? 'warning' : 'info'
}

async function load() {
  loading.value = true
  try {
    const { data } = await http.get('/users')
    users.value = data
  } finally {
    loading.value = false
  }
}

async function create() {
  if (!dialog.username || !dialog.password) {
    ElMessage.warning('Benutzername und Passwort erforderlich')
    return
  }
  await http.post('/users', { username: dialog.username, password: dialog.password, role: dialog.role })
  ElMessage.success('Benutzer erstellt')
  dialog.visible = false
  dialog.username = ''
  dialog.password = ''
  dialog.role = 'viewer'
  await load()
}

async function setRole(row: any, role: string) {
  if (role === row.role) return
  await http.post(`/users/${row.id}/role`, { role })
  ElMessage.success('Rolle geändert')
  await load()
}

async function resetPassword(row: any) {
  const { value } = await ElMessageBox.prompt(`Neues Passwort für ${row.username}`, 'Passwort zurücksetzen', {
    inputType: 'password',
  })
  await http.post(`/users/${row.id}/password`, { password: value })
  ElMessage.success('Passwort zurückgesetzt')
}

async function remove(row: any) {
  await ElMessageBox.confirm(`Benutzer "${row.username}" löschen?`, 'Bestätigen', { type: 'warning' })
  await http.delete(`/users/${row.id}`)
  ElMessage.success('Gelöscht')
  await load()
}

onMounted(load)
</script>
