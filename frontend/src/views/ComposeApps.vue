<template>
  <div>
    <h2 class="page-title">App-Stacks (Docker Compose)</h2>
    <el-alert v-if="data && !data.available" :closable="false" type="info" show-icon>
      Docker ist nicht installiert.
    </el-alert>

    <template v-else>
      <div class="toolbar">
        <el-button type="primary" @click="openDeploy">Stack deployen</el-button>
        <el-button @click="load">Aktualisieren</el-button>
      </div>

      <el-table :data="data?.apps || []" v-loading="loading" size="small">
        <el-table-column prop="name" label="Name" />
        <el-table-column prop="dir" label="Verzeichnis" />
        <el-table-column label="Aktionen" width="260">
          <template #default="{ row }">
            <el-button link @click="status(row)">Status</el-button>
            <el-button link type="danger" @click="down(row)">Stoppen & entfernen</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <el-dialog v-model="dialog.visible" title="Compose-Stack deployen" width="70%">
      <el-form label-width="80px">
        <el-form-item label="Name"><el-input v-model="dialog.name" placeholder="myapp" /></el-form-item>
        <el-form-item label="YAML">
          <el-input
            v-model="dialog.content"
            type="textarea"
            :rows="16"
            style="font-family: monospace"
            placeholder="services:&#10;  web:&#10;    image: nginx:latest&#10;    ports:&#10;      - '8080:80'"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" :loading="deploying" @click="deploy">Deployen</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="statusDialog.visible" :title="`Status: ${statusDialog.name}`" width="70%">
      <pre style="overflow: auto; max-height: 480px">{{ statusDialog.output }}</pre>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/client'

const data = ref<any>(null)
const loading = ref(false)
const deploying = ref(false)
const dialog = reactive({ visible: false, name: '', content: '' })
const statusDialog = reactive({ visible: false, name: '', output: '' })

async function load() {
  loading.value = true
  try {
    const res = await http.get('/compose')
    data.value = res.data
  } finally {
    loading.value = false
  }
}

function openDeploy() {
  dialog.name = ''
  dialog.content = ''
  dialog.visible = true
}

async function deploy() {
  if (!dialog.name || !dialog.content) {
    ElMessage.warning('Name und YAML erforderlich')
    return
  }
  deploying.value = true
  try {
    await http.post('/compose', { name: dialog.name, content: dialog.content })
    ElMessage.success('Stack deployed')
    dialog.visible = false
    await load()
  } finally {
    deploying.value = false
  }
}

async function status(row: any) {
  const { data: res } = await http.get(`/compose/${row.id}/status`)
  statusDialog.name = row.name
  statusDialog.output = res.status
  statusDialog.visible = true
}

async function down(row: any) {
  await ElMessageBox.confirm(`Stack "${row.name}" stoppen und entfernen?`, 'Bestätigen', { type: 'warning' })
  await http.post(`/compose/${row.id}/down`)
  ElMessage.success('Entfernt')
  await load()
}

onMounted(load)
</script>
