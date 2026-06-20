<template>
  <div>
    <h2 class="page-title">Datei-Manager</h2>

    <div class="toolbar">
      <el-input v-model="currentPath" style="width: 420px" @keyup.enter="list()">
        <template #prepend><el-icon><Folder /></el-icon></template>
      </el-input>
      <el-button @click="list()">Öffnen</el-button>
      <el-button @click="goUp">Übergeordnet</el-button>
      <div style="flex: 1" />
      <el-button type="primary" @click="newDialog.visible = true">Neue Datei</el-button>
      <el-button @click="mkdirDialog.visible = true">Neuer Ordner</el-button>
      <el-upload :show-file-list="false" :http-request="uploadFile">
        <el-button>Hochladen</el-button>
      </el-upload>
    </div>

    <el-table :data="entries" v-loading="loading" size="small">
      <el-table-column label="Name">
        <template #default="{ row }">
          <a v-if="row.is_dir" href="#" @click.prevent="enter(row)"><el-icon><Folder /></el-icon> {{ row.name }}</a>
          <span v-else><el-icon><Document /></el-icon> {{ row.name }}</span>
        </template>
      </el-table-column>
      <el-table-column label="Größe" width="120">
        <template #default="{ row }">{{ row.is_dir ? '-' : fmt(row.size) }}</template>
      </el-table-column>
      <el-table-column prop="mode" label="Rechte" width="100" />
      <el-table-column label="Geändert" width="180">
        <template #default="{ row }">{{ time(row.mod_time) }}</template>
      </el-table-column>
      <el-table-column label="Aktionen" width="320">
        <template #default="{ row }">
          <el-button v-if="!row.is_dir" link type="primary" @click="edit(row)">Bearbeiten</el-button>
          <el-button v-if="!row.is_dir" link @click="download(row)">Download</el-button>
          <el-button link @click="chmod(row)">Rechte</el-button>
          <el-button link @click="rename(row)">Umbenennen</el-button>
          <el-button link type="danger" @click="remove(row)">Löschen</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="editor.visible" :title="editor.path" width="70%">
      <el-input v-model="editor.content" type="textarea" :rows="20" style="font-family: monospace" />
      <template #footer>
        <el-button @click="editor.visible = false">Abbrechen</el-button>
        <el-button type="primary" @click="saveEdit">Speichern</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="newDialog.visible" title="Neue Datei" width="420px">
      <el-input v-model="newDialog.name" placeholder="dateiname.txt" />
      <template #footer>
        <el-button @click="newDialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" @click="createFile">Erstellen</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="mkdirDialog.visible" title="Neuer Ordner" width="420px">
      <el-input v-model="mkdirDialog.name" placeholder="ordnername" />
      <template #footer>
        <el-button @click="mkdirDialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" @click="createDir">Erstellen</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox, type UploadRequestOptions } from 'element-plus'
import http, { getToken } from '../api/client'
import { formatBytes, formatTime } from '../utils/format'

interface Entry {
  name: string
  path: string
  is_dir: boolean
  size: number
  mode: string
  mod_time: number
}

const currentPath = ref('/')
const entries = ref<Entry[]>([])
const loading = ref(false)
const fmt = formatBytes
const time = formatTime

const editor = reactive({ visible: false, path: '', content: '' })
const newDialog = reactive({ visible: false, name: '' })
const mkdirDialog = reactive({ visible: false, name: '' })

function join(dir: string, name: string): string {
  return dir.endsWith('/') ? dir + name : dir + '/' + name
}

async function list() {
  loading.value = true
  try {
    const { data } = await http.get('/files/list', { params: { path: currentPath.value } })
    currentPath.value = data.path
    entries.value = data.entries
  } finally {
    loading.value = false
  }
}

function enter(row: Entry) {
  currentPath.value = row.path
  list()
}

function goUp() {
  const p = currentPath.value.replace(/\/+$/, '')
  const idx = p.lastIndexOf('/')
  currentPath.value = idx <= 0 ? '/' : p.slice(0, idx)
  list()
}

async function edit(row: Entry) {
  const { data } = await http.get('/files/read', { params: { path: row.path } })
  editor.path = data.path
  editor.content = data.content
  editor.visible = true
}

async function saveEdit() {
  await http.post('/files/write', { path: editor.path, content: editor.content })
  ElMessage.success('Gespeichert')
  editor.visible = false
  list()
}

function download(row: Entry) {
  const token = getToken()
  window.open(`/api/files/download?path=${encodeURIComponent(row.path)}&token=${token}`, '_blank')
}

async function chmod(row: Entry) {
  const { value } = await ElMessageBox.prompt('Oktale Rechte (z.B. 0755)', 'Rechte ändern', {
    inputValue: row.mode,
  })
  await http.post('/files/chmod', { path: row.path, mode: value })
  ElMessage.success('Rechte geändert')
  list()
}

async function rename(row: Entry) {
  const { value } = await ElMessageBox.prompt('Neuer Pfad', 'Umbenennen', { inputValue: row.path })
  await http.post('/files/rename', { from: row.path, to: value })
  ElMessage.success('Umbenannt')
  list()
}

async function remove(row: Entry) {
  await ElMessageBox.confirm(`"${row.name}" wirklich löschen?`, 'Bestätigen', { type: 'warning' })
  await http.post('/files/delete', { path: row.path })
  ElMessage.success('Gelöscht')
  list()
}

async function createFile() {
  await http.post('/files/write', { path: join(currentPath.value, newDialog.name), content: '' })
  newDialog.visible = false
  newDialog.name = ''
  list()
}

async function createDir() {
  await http.post('/files/mkdir', { path: join(currentPath.value, mkdirDialog.name) })
  mkdirDialog.visible = false
  mkdirDialog.name = ''
  list()
}

async function uploadFile(options: UploadRequestOptions) {
  const form = new FormData()
  form.append('file', options.file)
  form.append('path', currentPath.value)
  await http.post('/files/upload', form)
  ElMessage.success('Hochgeladen')
  list()
}

onMounted(list)
</script>
