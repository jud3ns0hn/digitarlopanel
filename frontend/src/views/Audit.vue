<template>
  <div>
    <h2 class="page-title">Audit-Log</h2>
    <el-button class="toolbar" @click="load">Aktualisieren</el-button>
    <el-table :data="logs" v-loading="loading" size="small">
      <el-table-column label="Zeit" width="180">
        <template #default="{ row }">{{ new Date(row.created_at).toLocaleString() }}</template>
      </el-table-column>
      <el-table-column prop="username" label="Benutzer" width="140" />
      <el-table-column prop="action" label="Aktion" width="180" />
      <el-table-column prop="detail" label="Detail" />
      <el-table-column prop="ip" label="IP" width="140" />
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import http from '../api/client'

const logs = ref<any[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const { data } = await http.get('/audit')
    logs.value = data
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
