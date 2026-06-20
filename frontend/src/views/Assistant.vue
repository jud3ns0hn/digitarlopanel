<template>
  <div class="assistant">
    <h2 class="page-title">KI-Assistent</h2>

    <el-alert v-if="status && !status.configured" :closable="false" type="warning" show-icon style="margin-bottom: 12px">
      Kein KI-Provider konfiguriert. Unter <b>Einstellungen → KI-Assistent</b> Provider, Modell und API-Key setzen
      (Anthropic/Claude, OpenAI oder Ollama self-hosted).
    </el-alert>
    <el-alert v-else-if="status" :closable="false" type="info" show-icon style="margin-bottom: 12px">
      Provider: <b>{{ status.provider }}</b> · Modell: <b>{{ status.model }}</b>
    </el-alert>

    <el-card shadow="never" class="chat-card">
      <div ref="scrollEl" class="messages">
        <div v-for="(m, i) in messages" :key="i" :class="['msg', m.role]">
          <div class="bubble">
            <div v-if="m.role === 'assistant' && m.steps && m.steps.length" class="steps">
              <el-tag v-for="(st, j) in m.steps" :key="j" size="small" :type="st.error ? 'danger' : 'info'" effect="plain">
                {{ st.tool }}
              </el-tag>
            </div>
            <div class="content">{{ m.content }}</div>
          </div>
        </div>
        <div v-if="loading" class="msg assistant"><div class="bubble"><el-icon class="is-loading"><Loading /></el-icon> denkt nach …</div></div>
      </div>
    </el-card>

    <div class="composer">
      <el-input
        v-model="input"
        type="textarea"
        :rows="2"
        placeholder="Frag den Assistenten, z.B. „Wie ist die CPU-Auslastung und läuft Nginx?“"
        @keydown.enter.exact.prevent="send"
      />
      <el-button type="primary" :loading="loading" :disabled="!status?.configured" @click="send">Senden</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api/client'

interface Step { tool: string; error?: string }
interface Msg { role: 'user' | 'assistant'; content: string; steps?: Step[] }

const status = ref<any>(null)
const messages = ref<Msg[]>([])
const input = ref('')
const loading = ref(false)
const scrollEl = ref<HTMLElement>()

async function loadStatus() {
  const { data } = await http.get('/ai/status')
  status.value = data
}

async function scrollDown() {
  await nextTick()
  if (scrollEl.value) scrollEl.value.scrollTop = scrollEl.value.scrollHeight
}

async function send() {
  const text = input.value.trim()
  if (!text || loading.value) return
  messages.value.push({ role: 'user', content: text })
  input.value = ''
  loading.value = true
  await scrollDown()
  try {
    const payload = messages.value.map((m) => ({ role: m.role, content: m.content }))
    const { data } = await http.post('/ai/chat', { messages: payload })
    messages.value.push({ role: 'assistant', content: data.reply, steps: data.steps || [] })
  } catch (e: any) {
    messages.value.push({ role: 'assistant', content: 'Fehler: ' + (e.response?.data?.error || e.message) })
    ElMessage.error('Anfrage fehlgeschlagen')
  } finally {
    loading.value = false
    await scrollDown()
  }
}

onMounted(loadStatus)
</script>

<style scoped>
.assistant {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.chat-card {
  flex: 1;
  overflow: hidden;
}
.messages {
  height: 56vh;
  overflow-y: auto;
  padding: 4px;
}
.msg {
  display: flex;
  margin-bottom: 10px;
}
.msg.user {
  justify-content: flex-end;
}
.bubble {
  max-width: 78%;
  padding: 10px 12px;
  border-radius: 10px;
  background: #f0f2f5;
  white-space: pre-wrap;
  word-break: break-word;
}
.msg.user .bubble {
  background: #409eff;
  color: #fff;
}
.steps {
  margin-bottom: 6px;
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}
.composer {
  display: flex;
  gap: 8px;
  margin-top: 12px;
  align-items: flex-end;
}
</style>
