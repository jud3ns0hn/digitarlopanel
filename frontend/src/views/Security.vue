<template>
  <div>
    <h2 class="page-title">Sicherheit</h2>

    <el-tabs v-model="tab">
      <!-- Fail2ban -->
      <el-tab-pane label="Fail2ban" name="fail2ban">
        <div class="toolbar">
          <el-button @click="loadFail2ban">Aktualisieren</el-button>
          <el-button v-if="!f2b.available" type="primary" :loading="installing.f2b" @click="install('fail2ban')">
            Installieren
          </el-button>
        </div>
        <el-alert v-if="!f2b.available" :closable="false" type="info" show-icon>
          Fail2ban ist nicht installiert. Es blockiert IPs nach fehlgeschlagenen Login-Versuchen.
        </el-alert>
        <template v-else>
          <el-table :data="f2b.jails" size="small">
            <el-table-column label="Jail">
              <template #default="{ row }">{{ row }}</template>
            </el-table-column>
            <el-table-column label="Aktionen" width="160">
              <template #default="{ row }">
                <el-button link @click="openJail(row)">Details</el-button>
              </template>
            </el-table-column>
          </el-table>
        </template>

        <el-dialog v-model="jailDialog.visible" :title="`Jail: ${jailDialog.name}`" width="560px">
          <div class="toolbar">
            <el-input v-model="jailDialog.banIP" placeholder="IP für Bann/Entbann" style="width: 220px" />
            <el-button type="danger" plain @click="ban(true)">Bannen</el-button>
            <el-button type="success" plain @click="ban(false)">Entbannen</el-button>
          </div>
          <p>Gebannte IPs:</p>
          <el-tag v-for="ip in jailDialog.banned" :key="ip" style="margin: 2px">{{ ip }}</el-tag>
          <el-empty v-if="!jailDialog.banned.length" description="Keine gebannten IPs" :image-size="60" />
        </el-dialog>
      </el-tab-pane>

      <!-- Supervisor -->
      <el-tab-pane label="Supervisor" name="supervisor">
        <div class="toolbar">
          <el-button @click="loadSupervisor">Aktualisieren</el-button>
          <el-button v-if="!sup.available" type="primary" :loading="installing.sup" @click="install('supervisor')">
            Installieren
          </el-button>
        </div>
        <el-alert v-if="!sup.available" :closable="false" type="info" show-icon>
          Supervisor verwaltet langlaufende Prozesse (Worker, Daemons).
        </el-alert>
        <el-table v-else :data="sup.processes" size="small">
          <el-table-column prop="name" label="Programm" />
          <el-table-column label="Status" width="120">
            <template #default="{ row }">
              <el-tag :type="row.status === 'RUNNING' ? 'success' : 'info'" size="small">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="detail" label="Detail" />
          <el-table-column label="Aktionen" width="220">
            <template #default="{ row }">
              <el-button link @click="supAction(row.name, 'start')">Start</el-button>
              <el-button link @click="supAction(row.name, 'stop')">Stop</el-button>
              <el-button link @click="supAction(row.name, 'restart')">Neustart</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- ClamAV -->
      <el-tab-pane label="ClamAV" name="clamav">
        <div class="toolbar">
          <el-button @click="loadClamAV">Aktualisieren</el-button>
          <el-button v-if="!clam.available" type="primary" :loading="installing.clam" @click="install('clamav')">
            Installieren
          </el-button>
        </div>
        <el-alert v-if="!clam.available" :closable="false" type="info" show-icon>
          ClamAV durchsucht Dateien auf Schadsoftware.
        </el-alert>
        <template v-else>
          <el-tag type="success">{{ clam.version }}</el-tag>
          <el-form inline style="margin-top: 16px">
            <el-form-item label="Pfad">
              <el-input v-model="scanPath" placeholder="/var/www" style="width: 320px" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="scanning" @click="scan">Scan starten</el-button>
            </el-form-item>
          </el-form>
          <el-result
            v-if="scanResult"
            :icon="scanResult.clean ? 'success' : 'error'"
            :title="scanResult.clean ? 'Keine Bedrohungen gefunden' : `${scanResult.infected.length} Treffer`"
          >
            <template #sub-title>
              <div v-for="line in scanResult.infected" :key="line">{{ line }}</div>
            </template>
          </el-result>
        </template>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api/client'

const tab = ref('fail2ban')
const f2b = ref<any>({ available: false, jails: [] })
const sup = ref<any>({ available: false, processes: [] })
const clam = ref<any>({ available: false })
const installing = reactive({ f2b: false, sup: false, clam: false })
const scanPath = ref('/var/www')
const scanning = ref(false)
const scanResult = ref<any>(null)

const jailDialog = reactive<{ visible: boolean; name: string; banned: string[]; banIP: string }>({
  visible: false,
  name: '',
  banned: [],
  banIP: '',
})

async function loadFail2ban() {
  f2b.value = (await http.get('/fail2ban')).data
}
async function loadSupervisor() {
  sup.value = (await http.get('/supervisor')).data
}
async function loadClamAV() {
  clam.value = (await http.get('/clamav')).data
}

async function install(kind: 'fail2ban' | 'supervisor' | 'clamav') {
  const map = { fail2ban: 'f2b', supervisor: 'sup', clamav: 'clam' } as const
  installing[map[kind]] = true
  try {
    await http.post(`/${kind}/install`)
    ElMessage.success('Installiert')
    if (kind === 'fail2ban') await loadFail2ban()
    else if (kind === 'supervisor') await loadSupervisor()
    else await loadClamAV()
  } finally {
    installing[map[kind]] = false
  }
}

async function openJail(name: string) {
  jailDialog.name = name
  jailDialog.banIP = ''
  const res = await http.get('/fail2ban/jail', { params: { name } })
  jailDialog.banned = res.data.banned || []
  jailDialog.visible = true
}

async function ban(doBan: boolean) {
  if (!jailDialog.banIP) {
    ElMessage.warning('IP eingeben')
    return
  }
  const url = doBan ? '/fail2ban/ban' : '/fail2ban/unban'
  await http.post(url, { jail: jailDialog.name, ip: jailDialog.banIP })
  ElMessage.success(doBan ? 'Gebannt' : 'Entbannt')
  await openJail(jailDialog.name)
}

async function supAction(name: string, action: string) {
  await http.post('/supervisor/action', { name, action })
  ElMessage.success(`${action} ausgeführt`)
  await loadSupervisor()
}

async function scan() {
  scanning.value = true
  scanResult.value = null
  try {
    scanResult.value = (await http.post('/clamav/scan', { path: scanPath.value })).data
  } finally {
    scanning.value = false
  }
}

onMounted(() => {
  loadFail2ban()
  loadSupervisor()
  loadClamAV()
})
</script>
