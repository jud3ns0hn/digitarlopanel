<template>
  <div>
    <h2 class="page-title">Firewall</h2>
    <el-alert v-if="data" :closable="false" type="info" style="margin-bottom: 16px">
      Backend: <b>{{ data.backend }}</b>
      <span v-if="data.error"> · {{ data.error }}</span>
    </el-alert>

    <el-card shadow="never" style="margin-bottom: 16px">
      <template #header>Port freigeben</template>
      <div class="toolbar" style="margin: 0">
        <el-input-number v-model="form.port" :min="1" :max="65535" />
        <el-select v-model="form.proto" style="width: 100px">
          <el-option label="tcp" value="tcp" />
          <el-option label="udp" value="udp" />
        </el-select>
        <el-button type="primary" @click="allow">Freigeben</el-button>
      </div>
    </el-card>

    <el-card shadow="never">
      <template #header>Freigegebene Ports</template>
      <el-table :data="data?.rules || []" size="small">
        <el-table-column prop="port" label="Port" width="120" />
        <el-table-column prop="proto" label="Protokoll" width="120" />
        <el-table-column prop="raw" label="Regel" />
        <el-table-column label="Aktionen" width="120">
          <template #default="{ row }">
            <el-button link type="danger" @click="deny(row)">Entfernen</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api/client'

const data = ref<any>(null)
const form = reactive({ port: 80, proto: 'tcp' })

async function load() {
  const res = await http.get('/firewall')
  data.value = res.data
}

async function allow() {
  await http.post('/firewall/allow', { port: form.port, proto: form.proto })
  ElMessage.success(`Port ${form.port}/${form.proto} freigegeben`)
  await load()
}

async function deny(row: any) {
  await http.post('/firewall/deny', { port: row.port, proto: row.proto })
  ElMessage.success(`Port ${row.port}/${row.proto} entfernt`)
  await load()
}

onMounted(load)
</script>
