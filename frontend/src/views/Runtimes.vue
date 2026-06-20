<template>
  <div>
    <h2 class="page-title">Laufzeitumgebungen</h2>
    <div class="toolbar">
      <el-button @click="load">Aktualisieren</el-button>
      <el-tag type="info">Paketquelle: {{ family }}</el-tag>
    </div>

    <el-table :data="runtimes" v-loading="loading" size="small">
      <el-table-column prop="name" label="Laufzeit" width="200" />
      <el-table-column prop="description" label="Beschreibung" />
      <el-table-column label="Status" width="160">
        <template #default="{ row }">
          <el-tag v-if="row.installed" type="success" size="small">{{ row.version || 'installiert' }}</el-tag>
          <el-tag v-else type="info" size="small">nicht installiert</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Aktion" width="160">
        <template #default="{ row }">
          <el-button
            v-if="!row.installed"
            type="primary"
            size="small"
            :loading="installing === row.key"
            @click="install(row)"
          >
            Installieren
          </el-button>
          <span v-else style="color: #67c23a">✓</span>
        </template>
      </el-table-column>
    </el-table>

    <el-alert :closable="false" type="info" show-icon style="margin-top: 16px">
      Laufzeiten werden über den System-Paketmanager installiert (apt bzw. dnf/yum).
      Für aktuellere Versionen empfiehlt sich zusätzlich ein Versionsmanager (nvm, pyenv) im Terminal.
    </el-alert>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api/client'

const runtimes = ref<any[]>([])
const family = ref('')
const loading = ref(false)
const installing = ref<string | null>(null)

async function load() {
  loading.value = true
  try {
    const { data } = await http.get('/runtimes')
    runtimes.value = data.runtimes || []
    family.value = data.family || ''
  } finally {
    loading.value = false
  }
}

async function install(row: any) {
  installing.value = row.key
  try {
    await http.post('/runtimes/install', { key: row.key })
    ElMessage.success(`${row.name} installiert`)
    await load()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || 'Installation fehlgeschlagen')
  } finally {
    installing.value = null
  }
}

onMounted(load)
</script>
