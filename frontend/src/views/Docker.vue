<template>
  <div>
    <h2 class="page-title">Docker</h2>

    <el-alert v-if="data && !data.available" :closable="false" type="info" show-icon>
      Docker ist auf diesem Server nicht installiert. Über „Software“ oder die Shell installierbar.
    </el-alert>

    <template v-else>
      <div class="toolbar">
        <el-input v-model="pullImage" placeholder="Image ziehen, z.B. nginx:latest" style="width: 320px" />
        <el-button type="primary" :loading="pulling" @click="pull">Pull</el-button>
        <el-button @click="load">Aktualisieren</el-button>
      </div>

      <el-card shadow="never" style="margin-bottom: 16px">
        <template #header>Container</template>
        <el-table :data="data?.containers || []" size="small">
          <el-table-column prop="name" label="Name" />
          <el-table-column prop="image" label="Image" />
          <el-table-column label="Status" width="130">
            <template #default="{ row }">
              <el-tag :type="row.state === 'running' ? 'success' : 'info'">{{ row.state }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="Detail" />
          <el-table-column prop="ports" label="Ports" />
          <el-table-column label="Aktionen" width="300">
            <template #default="{ row }">
              <el-button link type="success" @click="action(row, 'start')">Start</el-button>
              <el-button link @click="action(row, 'stop')">Stop</el-button>
              <el-button link @click="action(row, 'restart')">Neustart</el-button>
              <el-button link type="danger" @click="remove(row)">Entfernen</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>

      <el-card shadow="never">
        <template #header>Images</template>
        <el-table :data="data?.images || []" size="small">
          <el-table-column prop="repository" label="Repository" />
          <el-table-column prop="tag" label="Tag" width="160" />
          <el-table-column prop="size" label="Größe" width="120" />
        </el-table>
      </el-card>
    </template>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/client'

const data = ref<any>(null)
const pullImage = ref('')
const pulling = ref(false)

async function load() {
  const res = await http.get('/docker')
  data.value = res.data
}

async function action(row: any, act: string) {
  await http.post('/docker/container', { id: row.id, action: act })
  ElMessage.success(`${row.name}: ${act}`)
  await load()
}

async function remove(row: any) {
  await ElMessageBox.confirm(`Container "${row.name}" entfernen?`, 'Bestätigen', { type: 'warning' })
  await http.post('/docker/container', { id: row.id, action: 'remove' })
  ElMessage.success('Entfernt')
  await load()
}

async function pull() {
  if (!pullImage.value) return
  pulling.value = true
  try {
    await http.post('/docker/pull', { image: pullImage.value })
    ElMessage.success('Image geladen')
    pullImage.value = ''
    await load()
  } finally {
    pulling.value = false
  }
}

onMounted(load)
</script>
