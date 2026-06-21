<template>
  <div>
    <div class="page-head">
      <h2 class="page-title">Websites</h2>
      <div class="head-actions">
        <el-tag v-if="phpVersions.length" type="success" effect="plain" size="small">
          PHP: {{ phpVersions.map((v) => v.version).join(', ') }}
        </el-tag>
        <el-tag v-else type="warning" effect="plain" size="small">Kein PHP-FPM</el-tag>
        <el-button :icon="Refresh" circle @click="load" />
        <el-button type="primary" :icon="Plus" @click="dialog.visible = true">Neue Website</el-button>
      </div>
    </div>

    <el-empty v-if="!loading && !sites.length" description="Noch keine Websites angelegt">
      <el-button type="primary" @click="dialog.visible = true">Erste Website anlegen</el-button>
    </el-empty>

    <el-table v-else :data="sites" v-loading="loading" size="default" @row-click="openDetail">
      <el-table-column prop="domain" label="Domain" min-width="200">
        <template #default="{ row }">
          <a :href="`http://${row.domain}`" target="_blank" class="domain-link" @click.stop>
            <el-icon><Link /></el-icon> {{ row.domain }}
          </a>
        </template>
      </el-table-column>
      <el-table-column label="Typ" width="170">
        <template #default="{ row }">
          <el-tag v-if="row.proxy_pass" type="warning" size="small" effect="light">Reverse-Proxy</el-tag>
          <el-tag v-else-if="row.redirect" type="info" size="small" effect="light">Redirect</el-tag>
          <el-tag v-else-if="row.php_version" type="primary" size="small" effect="light">PHP {{ row.php_version }}</el-tag>
          <el-tag v-else size="small" effect="light">Statisch</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Schutz" width="150">
        <template #default="{ row }">
          <el-tooltip v-if="row.waf" content="WAF aktiv"><el-icon color="#67c23a"><CircleCheck /></el-icon></el-tooltip>
          <el-tooltip v-if="row.basic_auth_user" content="Basic-Auth"><el-icon color="#e6a23c"><Lock /></el-icon></el-tooltip>
          <span v-if="!row.waf && !row.basic_auth_user" style="color: #c0c4cc">–</span>
        </template>
      </el-table-column>
      <el-table-column label="Status" width="110">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'" effect="dark" size="small">
            {{ row.enabled ? 'aktiv' : 'inaktiv' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Aktionen" width="240" align="right">
        <template #default="{ row }">
          <el-button link type="primary" @click.stop="openDetail(row)">Verwalten</el-button>
          <el-button link @click.stop="toggle(row)">{{ row.enabled ? 'Stoppen' : 'Starten' }}</el-button>
          <el-button link type="danger" @click.stop="remove(row)">Löschen</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Create dialog -->
    <el-dialog v-model="dialog.visible" title="Neue Website" width="520px">
      <el-form label-width="130px">
        <el-form-item label="Domain"><el-input v-model="dialog.domain" placeholder="example.com" /></el-form-item>
        <el-form-item label="Document-Root">
          <el-input v-model="dialog.root" placeholder="/var/www/example.com" />
        </el-form-item>
        <el-form-item label="PHP-Version">
          <el-select v-model="dialog.php_version" style="width: 100%" placeholder="Kein PHP (statisch)">
            <el-option label="Kein PHP (statisch)" value="" />
            <el-option v-for="v in phpVersions" :key="v.version" :label="`PHP ${v.version}`" :value="v.version" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" :loading="saving" @click="create">Erstellen</el-button>
      </template>
    </el-dialog>

    <!-- Detail drawer -->
    <el-drawer v-model="drawer.visible" :title="drawer.site?.domain" size="620px" @open="loadDetail">
      <template v-if="drawer.site">
        <el-tabs v-model="drawer.tab">
          <!-- Overview -->
          <el-tab-pane label="Übersicht" name="overview">
            <el-descriptions :column="1" border size="small">
              <el-descriptions-item label="Domain">{{ drawer.site.domain }}</el-descriptions-item>
              <el-descriptions-item label="Document-Root">{{ drawer.site.root }}</el-descriptions-item>
              <el-descriptions-item label="Status">
                <el-tag :type="drawer.site.enabled ? 'success' : 'info'" size="small">
                  {{ drawer.site.enabled ? 'aktiv' : 'inaktiv' }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="Typ">
                <span v-if="drawer.site.proxy_pass">Reverse-Proxy → {{ drawer.site.proxy_pass }}</span>
                <span v-else-if="drawer.site.redirect">Redirect → {{ drawer.site.redirect }}</span>
                <span v-else-if="drawer.site.php_version">PHP {{ drawer.site.php_version }}</span>
                <span v-else>Statisch</span>
              </el-descriptions-item>
              <el-descriptions-item label="PHP-FPM-Socket" v-if="detail.php_socket">{{ detail.php_socket }}</el-descriptions-item>
              <el-descriptions-item label="SSL-Zertifikat">
                <template v-if="detail.certificate">
                  <el-tag :type="certDays >= 14 ? 'success' : 'danger'" size="small">
                    {{ detail.certificate.type }} · {{ certDays }} Tage gültig
                  </el-tag>
                </template>
                <el-tag v-else type="info" size="small">kein Zertifikat</el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="WAF">
                <el-switch :model-value="drawer.site.waf" disabled />
              </el-descriptions-item>
              <el-descriptions-item label="Nginx-Config">
                <code style="font-size: 12px">{{ detail.config_path }}</code>
              </el-descriptions-item>
            </el-descriptions>

            <el-divider content-position="left">PHP-Version</el-divider>
            <div class="row-inline">
              <el-select v-model="phpEdit" placeholder="PHP wählen" style="width: 220px">
                <el-option label="Kein PHP (deaktivieren)" value="" />
                <el-option v-for="v in phpVersions" :key="v.version" :label="`PHP ${v.version}`" :value="v.version" />
              </el-select>
              <el-button type="primary" @click="savePHP">Übernehmen</el-button>
            </div>

            <el-divider content-position="left">Reverse-Proxy</el-divider>
            <div class="row-inline">
              <el-input v-model="proxyEdit" placeholder="http://127.0.0.1:3000 (leer = aus)" style="width: 320px" />
              <el-button type="primary" @click="saveProxy">Speichern</el-button>
            </div>
          </el-tab-pane>

          <!-- Logs -->
          <el-tab-pane label="Logs" name="logs">
            <div class="row-inline" style="margin-bottom: 10px">
              <el-radio-group v-model="logType" @change="loadLogs">
                <el-radio-button value="access">Access</el-radio-button>
                <el-radio-button value="error">Error</el-radio-button>
              </el-radio-group>
              <el-button :icon="Refresh" @click="loadLogs">Aktualisieren</el-button>
            </div>
            <pre class="logbox">{{ logText || 'Noch keine Log-Einträge.' }}</pre>
          </el-tab-pane>

          <!-- Config -->
          <el-tab-pane label="Konfiguration" name="config">
            <el-form label-width="160px">
              <el-form-item label="301-Redirect">
                <el-input v-model="cfg.redirect" placeholder="https://ziel.de (leer = aus)" />
              </el-form-item>
              <el-form-item label="WAF (Schutzregeln)">
                <el-switch v-model="cfg.waf" />
                <span class="hint">Blockiert SQL-Injection/XSS-Muster und bekannte Scanner.</span>
              </el-form-item>
              <el-form-item label="Basic-Auth Benutzer">
                <el-input v-model="cfg.basic_auth_user" placeholder="leer = kein Schutz" />
              </el-form-item>
              <el-form-item label="Basic-Auth Passwort">
                <el-input v-model="cfg.basic_auth_password" type="password" show-password placeholder="leer = unverändert" />
              </el-form-item>
              <el-form-item label="Eigene Nginx-Config">
                <el-input v-model="cfg.extra_config" type="textarea" :rows="5" placeholder="z.B. client_max_body_size 100m;" />
              </el-form-item>
              <el-form-item>
                <el-button type="primary" @click="saveConfig">Konfiguration speichern</el-button>
              </el-form-item>
            </el-form>
          </el-tab-pane>
        </el-tabs>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Plus, Link, Lock, CircleCheck } from '@element-plus/icons-vue'
import http from '../api/client'

const sites = ref<any[]>([])
const phpVersions = ref<any[]>([])
const loading = ref(false)
const saving = ref(false)
const dialog = reactive({ visible: false, domain: '', root: '', php_version: '' })

const drawer = reactive<{ visible: boolean; tab: string; site: any }>({ visible: false, tab: 'overview', site: null })
const detail = reactive<any>({ certificate: null, php_socket: '', config_path: '' })
const phpEdit = ref('')
const proxyEdit = ref('')
const logType = ref('access')
const logText = ref('')
const cfg = reactive({ redirect: '', waf: false, basic_auth_user: '', basic_auth_password: '', extra_config: '' })

const certDays = computed(() => {
  if (!detail.certificate?.not_after) return 0
  return Math.max(0, Math.round((new Date(detail.certificate.not_after).getTime() - Date.now()) / 86400000))
})

async function load() {
  loading.value = true
  try {
    const [s, p] = await Promise.all([http.get('/websites'), http.get('/php')])
    sites.value = s.data
    phpVersions.value = p.data.versions || []
  } finally {
    loading.value = false
  }
}

function openDetail(row: any) {
  drawer.site = row
  drawer.tab = 'overview'
  drawer.visible = true
}

async function loadDetail() {
  if (!drawer.site) return
  phpEdit.value = drawer.site.php_version || ''
  proxyEdit.value = drawer.site.proxy_pass || ''
  cfg.redirect = drawer.site.redirect || ''
  cfg.waf = !!drawer.site.waf
  cfg.basic_auth_user = drawer.site.basic_auth_user || ''
  cfg.basic_auth_password = ''
  cfg.extra_config = drawer.site.extra_config || ''
  const { data } = await http.get(`/websites/${drawer.site.id}`)
  detail.certificate = data.certificate || null
  detail.php_socket = data.php_socket || ''
  detail.config_path = data.config_path || ''
  await loadLogs()
}

async function loadLogs() {
  if (!drawer.site) return
  const { data } = await http.get(`/websites/${drawer.site.id}/logs`, { params: { type: logType.value, lines: 200 } })
  logText.value = data.content || ''
}

async function create() {
  if (!dialog.domain || !dialog.root) {
    ElMessage.warning('Domain und Root erforderlich')
    return
  }
  saving.value = true
  try {
    await http.post('/websites', { domain: dialog.domain, root: dialog.root, php_version: dialog.php_version })
    ElMessage.success('Website erstellt')
    dialog.visible = false
    dialog.domain = ''
    dialog.root = ''
    dialog.php_version = ''
    await load()
  } finally {
    saving.value = false
  }
}

async function savePHP() {
  await http.post(`/websites/${drawer.site.id}/php`, { version: phpEdit.value })
  ElMessage.success('PHP-Version aktualisiert')
  await refreshAfterEdit()
}

async function saveProxy() {
  await http.post(`/websites/${drawer.site.id}/proxy`, { proxy_pass: proxyEdit.value })
  ElMessage.success('Proxy aktualisiert')
  await refreshAfterEdit()
}

async function saveConfig() {
  await http.post(`/websites/${drawer.site.id}/config`, {
    redirect: cfg.redirect,
    waf: cfg.waf,
    basic_auth_user: cfg.basic_auth_user,
    basic_auth_password: cfg.basic_auth_password,
    extra_config: cfg.extra_config,
  })
  ElMessage.success('Konfiguration gespeichert')
  await refreshAfterEdit()
}

async function refreshAfterEdit() {
  await load()
  drawer.site = sites.value.find((x) => x.id === drawer.site.id) || drawer.site
  await loadDetail()
}

async function toggle(row: any) {
  await http.post(`/websites/${row.id}/toggle`)
  await load()
}

async function remove(row: any) {
  await ElMessageBox.confirm(`Website "${row.domain}" löschen?`, 'Bestätigen', { type: 'warning' })
  await http.delete(`/websites/${row.id}`)
  ElMessage.success('Gelöscht')
  if (drawer.site?.id === row.id) drawer.visible = false
  await load()
}

onMounted(load)
</script>

<style scoped>
.page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
.head-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}
.domain-link {
  color: var(--el-color-primary);
  text-decoration: none;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.domain-link:hover {
  text-decoration: underline;
}
.row-inline {
  display: flex;
  align-items: center;
  gap: 10px;
}
.hint {
  margin-left: 10px;
  color: #909399;
  font-size: 12px;
}
.logbox {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 12px;
  border-radius: 6px;
  font-size: 12px;
  line-height: 1.5;
  max-height: 460px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
