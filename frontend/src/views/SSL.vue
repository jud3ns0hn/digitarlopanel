<template>
  <div>
    <h2 class="page-title">SSL-Zertifikate</h2>
    <div class="toolbar">
      <el-button type="primary" @click="leDialog.visible = true">Let's Encrypt</el-button>
      <el-button @click="ssDialog.visible = true">Self-Signed</el-button>
      <el-button @click="load">Aktualisieren</el-button>
    </div>

    <el-table :data="certs" v-loading="loading" size="small">
      <el-table-column prop="domain" label="Domain" />
      <el-table-column label="Typ" width="140">
        <template #default="{ row }">
          <el-tag :type="row.type === 'letsencrypt' ? 'success' : 'warning'">{{ row.type }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Gültig bis" width="200">
        <template #default="{ row }">{{ row.not_after ? new Date(row.not_after).toLocaleDateString() : '—' }}</template>
      </el-table-column>
      <el-table-column label="Aktionen" width="140">
        <template #default="{ row }">
          <el-button link type="danger" @click="remove(row)">Löschen</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="leDialog.visible" title="Let's Encrypt Zertifikat" width="480px">
      <el-alert :closable="false" type="info" show-icon style="margin-bottom: 12px">
        Die Website muss bereits existieren und die Domain per DNS auf diesen Server zeigen (Port 80 erreichbar).
      </el-alert>
      <el-form label-width="100px">
        <el-form-item label="Domain"><el-input v-model="leDialog.domain" placeholder="example.com" /></el-form-item>
        <el-form-item label="E-Mail"><el-input v-model="leDialog.email" placeholder="admin@example.com" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="leDialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" :loading="issuing" @click="issue">Ausstellen</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="ssDialog.visible" title="Self-Signed Zertifikat" width="480px">
      <el-form label-width="100px">
        <el-form-item label="Domain"><el-input v-model="ssDialog.domain" placeholder="intern.example.com" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ssDialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" :loading="issuing" @click="selfSigned">Erstellen</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/client'

const certs = ref<any[]>([])
const loading = ref(false)
const issuing = ref(false)
const leDialog = reactive({ visible: false, domain: '', email: '' })
const ssDialog = reactive({ visible: false, domain: '' })

async function load() {
  loading.value = true
  try {
    const { data } = await http.get('/ssl')
    certs.value = data
  } finally {
    loading.value = false
  }
}

async function issue() {
  if (!leDialog.domain || !leDialog.email) {
    ElMessage.warning('Domain und E-Mail erforderlich')
    return
  }
  issuing.value = true
  try {
    await http.post('/ssl/issue', { domain: leDialog.domain, email: leDialog.email })
    ElMessage.success('Zertifikat ausgestellt')
    leDialog.visible = false
    await load()
  } finally {
    issuing.value = false
  }
}

async function selfSigned() {
  if (!ssDialog.domain) {
    ElMessage.warning('Domain erforderlich')
    return
  }
  issuing.value = true
  try {
    await http.post('/ssl/self-signed', { domain: ssDialog.domain })
    ElMessage.success('Self-Signed Zertifikat erstellt')
    ssDialog.visible = false
    await load()
  } finally {
    issuing.value = false
  }
}

async function remove(row: any) {
  await ElMessageBox.confirm(`Zertifikat für "${row.domain}" löschen?`, 'Bestätigen', { type: 'warning' })
  await http.delete(`/ssl/${row.id}`)
  ElMessage.success('Gelöscht')
  await load()
}

onMounted(load)
</script>
