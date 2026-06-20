<template>
  <div>
    <h2 class="page-title">Redis</h2>
    <el-alert v-if="info && !info.available" :closable="false" type="warning" show-icon style="margin-bottom: 12px">
      Redis nicht erreichbar (redis-cli). Über „Software"/„App-Store" installierbar.
    </el-alert>

    <el-row :gutter="16" v-else>
      <el-col :span="10">
        <el-card shadow="never">
          <template #header>Server-Info ({{ info?.dbsize }} Keys)</template>
          <pre class="info">{{ info?.info }}</pre>
        </el-card>
      </el-col>
      <el-col :span="14">
        <el-card shadow="never">
          <template #header>
            <div class="toolbar" style="margin: 0">
              <el-input v-model="pattern" placeholder="Muster, z.B. *" style="width: 180px" @keyup.enter="loadKeys" />
              <el-button @click="loadKeys">Suchen</el-button>
            </div>
          </template>
          <el-table :data="keys" size="small" height="380" @row-click="openKey">
            <el-table-column prop="0" label="Key" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-dialog v-model="dialog.visible" :title="dialog.key" width="520px">
      <p>Typ: <b>{{ dialog.type }}</b> · TTL: {{ dialog.ttl }}</p>
      <el-input v-model="dialog.value" type="textarea" :rows="6" />
      <template #footer>
        <el-button type="danger" @click="delKey">Löschen</el-button>
        <el-button type="primary" @click="setKey">Speichern</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api/client'

const info = ref<any>(null)
const pattern = ref('*')
const keys = ref<string[][]>([])
const dialog = reactive({ visible: false, key: '', type: '', value: '', ttl: '' })

async function load() {
  const { data } = await http.get('/redis/info')
  info.value = data
  if (data.available) loadKeys()
}
async function loadKeys() {
  const { data } = await http.get('/redis/keys', { params: { pattern: pattern.value } })
  keys.value = (data.keys || []).map((k: string) => [k])
}
async function openKey(row: string[]) {
  const { data } = await http.get('/redis/get', { params: { key: row[0] } })
  dialog.key = data.key
  dialog.type = data.type
  dialog.value = data.value
  dialog.ttl = data.ttl
  dialog.visible = true
}
async function setKey() {
  await http.post('/redis/set', { key: dialog.key, value: dialog.value })
  ElMessage.success('Gespeichert')
  dialog.visible = false
  await loadKeys()
}
async function delKey() {
  await http.delete('/redis/key', { params: { key: dialog.key } })
  ElMessage.success('Gelöscht')
  dialog.visible = false
  await loadKeys()
}
onMounted(load)
</script>

<style scoped>
.info { max-height: 380px; overflow: auto; font-size: 12px; margin: 0; }
</style>
