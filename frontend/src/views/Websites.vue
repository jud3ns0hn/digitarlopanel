<template>
  <div>
    <h2 class="page-title">Websites</h2>
    <div class="toolbar">
      <el-button type="primary" @click="dialog.visible = true">Neue Website</el-button>
      <el-button @click="load">Aktualisieren</el-button>
      <el-tag v-if="phpVersions.length" type="info">
        PHP installiert: {{ phpVersions.map((v) => v.version).join(', ') }}
      </el-tag>
      <el-tag v-else type="warning">Kein PHP-FPM erkannt</el-tag>
    </div>

    <el-table :data="sites" v-loading="loading" size="small">
      <el-table-column prop="domain" label="Domain" />
      <el-table-column prop="root" label="Document-Root" />
      <el-table-column label="Typ" width="200">
        <template #default="{ row }">
          <el-tag v-if="row.proxy_pass" type="warning" size="small">Proxy → {{ row.proxy_pass }}</el-tag>
          <el-tag v-else-if="row.php_version" size="small">PHP {{ row.php_version }}</el-tag>
          <el-tag v-else type="info" size="small">Statisch</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Status" width="110">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? 'aktiv' : 'inaktiv' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Aktionen" width="320">
        <template #default="{ row }">
          <el-button link @click="setPHP(row)">PHP</el-button>
          <el-button link @click="setProxy(row)">Proxy</el-button>
          <el-button link @click="setConfig(row)">Konfig</el-button>
          <el-button link @click="toggle(row)">{{ row.enabled ? 'Deaktivieren' : 'Aktivieren' }}</el-button>
          <el-button link type="danger" @click="remove(row)">Löschen</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialog.visible" title="Neue Website" width="480px">
      <el-form label-width="120px">
        <el-form-item label="Domain"><el-input v-model="dialog.domain" placeholder="example.com" /></el-form-item>
        <el-form-item label="Document-Root">
          <el-input v-model="dialog.root" placeholder="/var/www/example.com" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" :loading="saving" @click="create">Erstellen</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="proxyDialog.visible" title="Reverse-Proxy" width="460px">
      <el-form label-width="120px">
        <el-form-item label="Ziel-URL">
          <el-input v-model="proxyDialog.target" placeholder="http://127.0.0.1:3000 (leer = aus)" />
        </el-form-item>
      </el-form>
      <el-alert :closable="false" type="info" show-icon>
        Leitet alle Anfragen an die Ziel-URL weiter (inkl. WebSocket-Upgrade). Ideal für Apps/Docker hinter Nginx mit SSL.
      </el-alert>
      <template #footer>
        <el-button @click="proxyDialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" @click="saveProxy">Speichern</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="configDialog.visible" title="Erweiterte Konfiguration" width="560px">
      <el-form label-width="150px">
        <el-form-item label="301-Redirect">
          <el-input v-model="configDialog.redirect" placeholder="https://ziel.de (leer = aus)" />
        </el-form-item>
        <el-form-item label="WAF (Schutzregeln)">
          <el-switch v-model="configDialog.waf" />
          <span style="margin-left: 10px; color: #909399; font-size: 12px">
            Blockiert SQL-Injection-/XSS-Muster und bekannte Scanner.
          </span>
        </el-form-item>
        <el-form-item label="Basic-Auth Benutzer">
          <el-input v-model="configDialog.basic_auth_user" placeholder="leer = kein Schutz" />
        </el-form-item>
        <el-form-item label="Basic-Auth Passwort">
          <el-input v-model="configDialog.basic_auth_password" type="password" show-password placeholder="leer = unverändert" />
        </el-form-item>
        <el-form-item label="Eigene Nginx-Config">
          <el-input v-model="configDialog.extra_config" type="textarea" :rows="5" placeholder="z.B. client_max_body_size 100m;" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="configDialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" @click="saveConfig">Speichern</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="phpDialog.visible" title="PHP-Version zuweisen" width="420px">
      <el-form label-width="120px">
        <el-form-item label="Version">
          <el-select v-model="phpDialog.version" style="width: 100%">
            <el-option label="Kein PHP (deaktivieren)" value="" />
            <el-option v-for="v in phpVersions" :key="v.version" :label="`PHP ${v.version}`" :value="v.version" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="phpDialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" @click="savePHP">Speichern</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/client'

const sites = ref<any[]>([])
const phpVersions = ref<any[]>([])
const loading = ref(false)
const saving = ref(false)
const dialog = reactive({ visible: false, domain: '', root: '' })
const phpDialog = reactive({ visible: false, id: 0, version: '' })
const proxyDialog = reactive({ visible: false, id: 0, target: '' })
const configDialog = reactive({
  visible: false,
  id: 0,
  redirect: '',
  basic_auth_user: '',
  basic_auth_password: '',
  extra_config: '',
  waf: false,
})

function setConfig(row: any) {
  configDialog.id = row.id
  configDialog.redirect = row.redirect || ''
  configDialog.basic_auth_user = row.basic_auth_user || ''
  configDialog.basic_auth_password = ''
  configDialog.extra_config = row.extra_config || ''
  configDialog.waf = !!row.waf
  configDialog.visible = true
}

async function saveConfig() {
  await http.post(`/websites/${configDialog.id}/config`, {
    redirect: configDialog.redirect,
    basic_auth_user: configDialog.basic_auth_user,
    basic_auth_password: configDialog.basic_auth_password,
    extra_config: configDialog.extra_config,
    waf: configDialog.waf,
  })
  ElMessage.success('Konfiguration gespeichert')
  configDialog.visible = false
  await load()
}

function setProxy(row: any) {
  proxyDialog.id = row.id
  proxyDialog.target = row.proxy_pass || ''
  proxyDialog.visible = true
}

async function saveProxy() {
  await http.post(`/websites/${proxyDialog.id}/proxy`, { proxy_pass: proxyDialog.target })
  ElMessage.success('Proxy aktualisiert')
  proxyDialog.visible = false
  await load()
}

async function load() {
  loading.value = true
  try {
    const [sitesRes, phpRes] = await Promise.all([http.get('/websites'), http.get('/php')])
    sites.value = sitesRes.data
    phpVersions.value = phpRes.data.versions || []
  } finally {
    loading.value = false
  }
}

async function create() {
  if (!dialog.domain || !dialog.root) {
    ElMessage.warning('Domain und Root erforderlich')
    return
  }
  saving.value = true
  try {
    await http.post('/websites', { domain: dialog.domain, root: dialog.root })
    ElMessage.success('Website erstellt')
    dialog.visible = false
    dialog.domain = ''
    dialog.root = ''
    await load()
  } finally {
    saving.value = false
  }
}

function setPHP(row: any) {
  phpDialog.id = row.id
  phpDialog.version = row.php_version || ''
  phpDialog.visible = true
}

async function savePHP() {
  await http.post(`/websites/${phpDialog.id}/php`, { version: phpDialog.version })
  ElMessage.success('PHP-Version aktualisiert')
  phpDialog.visible = false
  await load()
}

async function toggle(row: any) {
  await http.post(`/websites/${row.id}/toggle`)
  await load()
}

async function remove(row: any) {
  await ElMessageBox.confirm(`Website "${row.domain}" löschen?`, 'Bestätigen', { type: 'warning' })
  await http.delete(`/websites/${row.id}`)
  ElMessage.success('Gelöscht')
  await load()
}

onMounted(load)
</script>
