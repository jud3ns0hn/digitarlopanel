<template>
  <div>
    <h2 class="page-title">Logs</h2>
    <div class="toolbar">
      <el-radio-group v-model="mode">
        <el-radio-button value="journal">systemd-Journal</el-radio-button>
        <el-radio-button value="file">Logdatei (/var/log)</el-radio-button>
      </el-radio-group>
      <el-input
        v-if="mode === 'journal'"
        v-model="unit"
        placeholder="z.B. nginx.service"
        style="width: 240px"
      />
      <el-input v-else v-model="path" placeholder="/var/log/syslog" style="width: 280px" />
      <el-input-number v-model="lines" :min="50" :max="2000" :step="50" />
      <el-button type="primary" @click="load">Laden</el-button>
    </div>

    <el-card shadow="never">
      <pre class="log-output">{{ content || '— keine Daten —' }}</pre>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import http from '../api/client'

const mode = ref('journal')
const unit = ref('nginx.service')
const path = ref('/var/log/syslog')
const lines = ref(200)
const content = ref('')

async function load() {
  if (mode.value === 'journal') {
    const { data } = await http.get('/logs/journal', { params: { unit: unit.value, lines: lines.value } })
    content.value = data.content
  } else {
    const { data } = await http.get('/logs/file', { params: { path: path.value, lines: lines.value } })
    content.value = data.is_dir ? (data.entries || []).join('\n') : data.content
  }
}
</script>

<style scoped>
.log-output {
  margin: 0;
  max-height: 600px;
  overflow: auto;
  font-family: monospace;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
