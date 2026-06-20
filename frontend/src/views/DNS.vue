<template>
  <div>
    <h2 class="page-title">DNS (BIND)</h2>
    <div class="toolbar">
      <el-button type="primary" @click="zoneDialog.visible = true">Neue Zone</el-button>
      <el-button @click="load">Aktualisieren</el-button>
    </div>

    <el-table :data="zones" v-loading="loading" size="small" row-key="id" :expand-row-keys="expanded" @expand-change="onExpand">
      <el-table-column type="expand">
        <template #default="{ row }">
          <div style="padding: 12px 24px">
            <div class="toolbar" style="margin-bottom: 8px">
              <strong>Records für {{ row.domain }}</strong>
              <el-button size="small" type="primary" @click="openRecord(row)">Record hinzufügen</el-button>
            </div>
            <el-table :data="row.records || []" size="small">
              <el-table-column prop="name" label="Name" width="160" />
              <el-table-column prop="type" label="Typ" width="90" />
              <el-table-column prop="value" label="Wert" />
              <el-table-column prop="ttl" label="TTL" width="90" />
              <el-table-column prop="priority" label="Prio" width="70" />
              <el-table-column label="" width="100">
                <template #default="{ row: rec }">
                  <el-button link type="danger" @click="removeRecord(row, rec)">Löschen</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="domain" label="Domain" />
      <el-table-column prop="serial" label="Serial" width="160" />
      <el-table-column label="Records" width="100">
        <template #default="{ row }">{{ (row.records || []).length }}</template>
      </el-table-column>
      <el-table-column label="Aktionen" width="120">
        <template #default="{ row }">
          <el-button link type="danger" @click="removeZone(row)">Löschen</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="zoneDialog.visible" title="Neue DNS-Zone" width="460px">
      <el-form label-width="140px">
        <el-form-item label="Domain"><el-input v-model="zoneDialog.domain" placeholder="example.com" /></el-form-item>
        <el-form-item label="Nameserver"><el-input v-model="zoneDialog.ns" placeholder="ns1.example.com (optional)" /></el-form-item>
        <el-form-item label="Admin-E-Mail"><el-input v-model="zoneDialog.admin" placeholder="hostmaster@example.com" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="zoneDialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" @click="createZone">Erstellen</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="recordDialog.visible" title="Record hinzufügen" width="460px">
      <el-form label-width="100px">
        <el-form-item label="Name"><el-input v-model="recordDialog.name" placeholder="@ oder www" /></el-form-item>
        <el-form-item label="Typ">
          <el-select v-model="recordDialog.type">
            <el-option v-for="t in ['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS']" :key="t" :label="t" :value="t" />
          </el-select>
        </el-form-item>
        <el-form-item label="Wert"><el-input v-model="recordDialog.value" /></el-form-item>
        <el-form-item label="TTL"><el-input-number v-model="recordDialog.ttl" :min="60" :max="604800" /></el-form-item>
        <el-form-item v-if="recordDialog.type === 'MX'" label="Priorität">
          <el-input-number v-model="recordDialog.priority" :min="0" :max="65535" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="recordDialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" @click="createRecord">Hinzufügen</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/client'

const zones = ref<any[]>([])
const loading = ref(false)
const expanded = ref<number[]>([])
const zoneDialog = reactive({ visible: false, domain: '', ns: '', admin: '' })
const recordDialog = reactive({ visible: false, zoneId: 0, name: '@', type: 'A', value: '', ttl: 3600, priority: 10 })

function onExpand(_row: any, rows: any[]) {
  expanded.value = rows.map((r) => r.id)
}

async function load() {
  loading.value = true
  try {
    const { data } = await http.get('/dns')
    zones.value = data
  } finally {
    loading.value = false
  }
}

async function createZone() {
  if (!zoneDialog.domain) {
    ElMessage.warning('Domain erforderlich')
    return
  }
  await http.post('/dns', { domain: zoneDialog.domain, ns: zoneDialog.ns, admin: zoneDialog.admin })
  ElMessage.success('Zone erstellt')
  zoneDialog.visible = false
  zoneDialog.domain = ''
  await load()
}

async function removeZone(row: any) {
  await ElMessageBox.confirm(`Zone "${row.domain}" löschen?`, 'Bestätigen', { type: 'warning' })
  await http.delete(`/dns/${row.id}`)
  ElMessage.success('Gelöscht')
  await load()
}

function openRecord(zone: any) {
  recordDialog.zoneId = zone.id
  recordDialog.name = '@'
  recordDialog.type = 'A'
  recordDialog.value = ''
  recordDialog.ttl = 3600
  recordDialog.priority = 10
  recordDialog.visible = true
}

async function createRecord() {
  if (!recordDialog.value) {
    ElMessage.warning('Wert erforderlich')
    return
  }
  await http.post(`/dns/${recordDialog.zoneId}/records`, {
    name: recordDialog.name,
    type: recordDialog.type,
    value: recordDialog.value,
    ttl: recordDialog.ttl,
    priority: recordDialog.priority,
  })
  ElMessage.success('Record hinzugefügt')
  recordDialog.visible = false
  await load()
}

async function removeRecord(zone: any, rec: any) {
  await http.delete(`/dns/${zone.id}/records/${rec.id}`)
  ElMessage.success('Record gelöscht')
  await load()
}

onMounted(load)
</script>
