<template>
  <div>
    <h2 class="page-title">Einstellungen</h2>

    <el-card shadow="never" style="max-width: 560px; margin-bottom: 16px">
      <template #header>Passwort ändern</template>
      <el-form label-width="180px">
        <el-form-item label="Aktuelles Passwort">
          <el-input v-model="pw.old" type="password" show-password />
        </el-form-item>
        <el-form-item label="Neues Passwort">
          <el-input v-model="pw.new" type="password" show-password />
        </el-form-item>
        <el-form-item label="Wiederholen">
          <el-input v-model="pw.confirm" type="password" show-password />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="pwSaving" @click="changePassword">Speichern</el-button>
        </el-form-item>
      </el-form>
      <el-alert :closable="false" type="info" show-icon>
        Mindestens 10 Zeichen, Buchstaben und Ziffern. Andere Sitzungen werden abgemeldet.
      </el-alert>
    </el-card>

    <el-card shadow="never" style="max-width: 560px">
      <template #header>Zwei-Faktor-Authentifizierung (TOTP)</template>

      <div v-if="auth.user?.two_fa_enabled">
        <el-tag type="success">aktiviert</el-tag>
        <el-form label-width="180px" style="margin-top: 16px">
          <el-form-item label="Passwort zum Deaktivieren">
            <el-input v-model="disablePassword" type="password" show-password />
          </el-form-item>
          <el-form-item>
            <el-button type="danger" @click="disable2FA">2FA deaktivieren</el-button>
          </el-form-item>
        </el-form>
      </div>

      <div v-else>
        <el-button v-if="!setup.secret" type="primary" @click="start2FA">2FA einrichten</el-button>
        <div v-else>
          <p>Scanne diesen Schlüssel in deiner Authenticator-App (z.B. Google Authenticator):</p>
          <el-input :model-value="setup.secret" readonly style="margin-bottom: 8px">
            <template #prepend>Secret</template>
          </el-input>
          <el-input :model-value="setup.url" readonly type="textarea" :rows="2" style="margin-bottom: 12px" />
          <el-form label-width="180px">
            <el-form-item label="Code aus der App">
              <el-input v-model="setup.code" maxlength="6" placeholder="6-stellig" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="enable2FA">Aktivieren</el-button>
            </el-form-item>
          </el-form>
        </div>
      </div>
    </el-card>

    <el-card v-if="auth.isAdmin && panel" shadow="never" style="max-width: 560px; margin-top: 16px">
      <template #header>Panel-Einstellungen</template>
      <el-form label-width="180px">
        <el-form-item label="Datei-Manager-Wurzel"><el-input v-model="panel.file_root" /></el-form-item>
        <el-form-item label="Backup-Verzeichnis"><el-input v-model="panel.backup_dir" /></el-form-item>
        <el-form-item label="FTP-Systembenutzer"><el-input v-model="panel.ftp_user" /></el-form-item>
        <el-form-item label="FTP-Systemgruppe"><el-input v-model="panel.ftp_group" /></el-form-item>
        <el-form-item label="Listen-Adresse"><el-input v-model="panel.listen" /></el-form-item>
        <el-form-item label="HTTPS (TLS) aktiv"><el-switch v-model="panel.tls_enabled" /></el-form-item>
        <el-form-item label="Self-Signed automatisch"><el-switch v-model="panel.tls_auto_self_signed" /></el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="panelSaving" @click="savePanel">Speichern</el-button>
        </el-form-item>
      </el-form>
      <el-alert :closable="false" type="warning" show-icon>
        Änderungen an Listen-Adresse und TLS werden erst nach einem Neustart des Panels wirksam.
      </el-alert>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http, { updateToken } from '../api/client'
import { useAuthStore } from '../store/auth'

const auth = useAuthStore()
const pw = reactive({ old: '', new: '', confirm: '' })
const pwSaving = ref(false)
const setup = reactive({ secret: '', url: '', code: '' })
const disablePassword = ref('')
const panel = ref<any>(null)
const panelSaving = ref(false)

onMounted(async () => {
  if (auth.isAdmin) {
    try {
      const { data } = await http.get('/settings')
      panel.value = data
    } catch {
      /* non-admin or error */
    }
  }
})

async function savePanel() {
  panelSaving.value = true
  try {
    const { data } = await http.post('/settings', {
      file_root: panel.value.file_root,
      backup_dir: panel.value.backup_dir,
      ftp_user: panel.value.ftp_user,
      ftp_group: panel.value.ftp_group,
      listen: panel.value.listen,
      tls_enabled: panel.value.tls_enabled,
      tls_auto_self_signed: panel.value.tls_auto_self_signed,
    })
    ElMessage.success(data.restart_required ? 'Gespeichert — Neustart für Listen/TLS nötig' : 'Gespeichert')
  } finally {
    panelSaving.value = false
  }
}

async function changePassword() {
  if (pw.new.length < 10) {
    ElMessage.warning('Neues Passwort muss mindestens 10 Zeichen haben')
    return
  }
  if (pw.new !== pw.confirm) {
    ElMessage.warning('Passwörter stimmen nicht überein')
    return
  }
  pwSaving.value = true
  try {
    const { data } = await http.post('/change-password', { old_password: pw.old, new_password: pw.new })
    if (data.token) updateToken(data.token)
    ElMessage.success('Passwort geändert')
    pw.old = ''
    pw.new = ''
    pw.confirm = ''
  } finally {
    pwSaving.value = false
  }
}

async function start2FA() {
  const { data } = await http.post('/2fa/setup')
  setup.secret = data.secret
  setup.url = data.otpauth_url
}

async function enable2FA() {
  await http.post('/2fa/enable', { code: setup.code })
  ElMessage.success('2FA aktiviert')
  setup.secret = ''
  setup.url = ''
  setup.code = ''
  await auth.fetchMe()
}

async function disable2FA() {
  await http.post('/2fa/disable', { password: disablePassword.value })
  ElMessage.success('2FA deaktiviert')
  disablePassword.value = ''
  await auth.fetchMe()
}
</script>
