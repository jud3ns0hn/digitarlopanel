<template>
  <div>
    <h2 class="page-title">Terminal</h2>
    <el-alert
      :closable="false"
      type="warning"
      show-icon
      style="margin-bottom: 12px"
      title="Voller Shell-Zugriff mit den Rechten des Panels (root). Nur für Administratoren. Jede Sitzung wird protokolliert."
    />
    <div ref="termEl" class="terminal"></div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { getToken } from '../api/client'

const termEl = ref<HTMLElement>()
let term: Terminal | null = null
let ws: WebSocket | null = null
let fit: FitAddon | null = null

function sendResize() {
  if (ws?.readyState === WebSocket.OPEN && term) {
    ws.send(JSON.stringify({ t: 'r', cols: term.cols, rows: term.rows }))
  }
}

function onResize() {
  fit?.fit()
  sendResize()
}

onMounted(() => {
  term = new Terminal({ cursorBlink: true, fontSize: 13, theme: { background: '#1e1e1e' } })
  fit = new FitAddon()
  term.loadAddon(fit)
  term.open(termEl.value!)
  fit.fit()

  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  ws = new WebSocket(`${proto}://${location.host}/api/terminal?token=${getToken()}`)

  ws.onopen = () => sendResize()
  ws.onmessage = (ev) => term?.write(typeof ev.data === 'string' ? ev.data : '')
  ws.onclose = () => term?.write('\r\n\x1b[31m[Verbindung geschlossen]\x1b[0m\r\n')

  term.onData((data) => {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ t: 'i', d: data }))
    }
  })

  window.addEventListener('resize', onResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  ws?.close()
  term?.dispose()
})
</script>

<style scoped>
.terminal {
  height: 600px;
  background: #1e1e1e;
  padding: 8px;
  border-radius: 4px;
}
</style>
