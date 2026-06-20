<template>
  <div>
    <h2 class="page-title">Systemdienste</h2>
    <div class="toolbar">
      <el-input v-model="filter" placeholder="Filtern..." style="width: 280px" clearable />
      <el-button @click="load">Aktualisieren</el-button>
    </div>

    <el-table :data="filtered" v-loading="loading" size="small" height="600">
      <el-table-column prop="name" label="Unit" min-width="220" />
      <el-table-column prop="active" label="Active" width="120">
        <template #default="{ row }">
          <el-tag :type="row.active === 'active' ? 'success' : 'info'">{{ row.active }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="sub" label="Sub" width="120" />
      <el-table-column prop="description" label="Beschreibung" min-width="240" />
      <el-table-column label="Aktionen" width="280">
        <template #default="{ row }">
          <el-button link type="success" @click="control(row, 'start')">Start</el-button>
          <el-button link @click="control(row, 'stop')">Stop</el-button>
          <el-button link @click="control(row, 'restart')">Neustart</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api/client'

const units = ref<any[]>([])
const loading = ref(false)
const filter = ref('')

const filtered = computed(() => {
  const f = filter.value.toLowerCase()
  if (!f) return units.value
  return units.value.filter((u) => u.name.toLowerCase().includes(f) || u.description.toLowerCase().includes(f))
})

async function load() {
  loading.value = true
  try {
    const { data } = await http.get('/services')
    units.value = data
  } finally {
    loading.value = false
  }
}

async function control(row: any, action: string) {
  await http.post('/services/control', { unit: row.name, action })
  ElMessage.success(`${row.name}: ${action}`)
  await load()
}

onMounted(load)
</script>
