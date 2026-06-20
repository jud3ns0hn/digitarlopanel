<template>
  <div>
    <h2 class="page-title">Mailserver</h2>
    <el-alert :closable="false" type="info" show-icon style="margin-bottom: 16px">
      Verwaltet virtuelle Postfix/Dovecot-Domains und -Postfächer. Setzt einen konfigurierten Postfix/Dovecot-Stack voraus.
    </el-alert>

    <el-row :gutter="16">
      <el-col :span="10">
        <el-card shadow="never">
          <template #header>
            <div style="display: flex; justify-content: space-between; align-items: center">
              <span>Domains</span>
              <el-button link type="primary" @click="domainDialog.visible = true">Hinzufügen</el-button>
            </div>
          </template>
          <el-table :data="domains" size="small">
            <el-table-column prop="domain" label="Domain" />
            <el-table-column label="" width="90">
              <template #default="{ row }">
                <el-button link type="danger" @click="removeDomain(row)">Löschen</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="14">
        <el-card shadow="never">
          <template #header>
            <div style="display: flex; justify-content: space-between; align-items: center">
              <span>Postfächer</span>
              <el-button link type="primary" @click="accountDialog.visible = true">Hinzufügen</el-button>
            </div>
          </template>
          <el-table :data="accounts" size="small">
            <el-table-column prop="address" label="Adresse" />
            <el-table-column prop="quota" label="Quota (MB)" width="120" />
            <el-table-column label="" width="90">
              <template #default="{ row }">
                <el-button link type="danger" @click="removeAccount(row)">Löschen</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-dialog v-model="domainDialog.visible" title="Mail-Domain" width="420px">
      <el-input v-model="domainDialog.domain" placeholder="example.com" />
      <template #footer>
        <el-button @click="domainDialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" @click="createDomain">Erstellen</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="accountDialog.visible" title="Postfach" width="460px">
      <el-form label-width="120px">
        <el-form-item label="Adresse"><el-input v-model="accountDialog.address" placeholder="user@example.com" /></el-form-item>
        <el-form-item label="Passwort"><el-input v-model="accountDialog.password" type="password" show-password /></el-form-item>
        <el-form-item label="Quota (MB)"><el-input-number v-model="accountDialog.quota" :min="0" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="accountDialog.visible = false">Abbrechen</el-button>
        <el-button type="primary" @click="createAccount">Erstellen</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/client'

const domains = ref<any[]>([])
const accounts = ref<any[]>([])
const domainDialog = reactive({ visible: false, domain: '' })
const accountDialog = reactive({ visible: false, address: '', password: '', quota: 0 })

async function load() {
  const { data } = await http.get('/mail')
  domains.value = data.domains || []
  accounts.value = data.accounts || []
}

async function createDomain() {
  await http.post('/mail/domains', { domain: domainDialog.domain })
  ElMessage.success('Domain erstellt')
  domainDialog.visible = false
  domainDialog.domain = ''
  await load()
}

async function removeDomain(row: any) {
  await ElMessageBox.confirm(`Domain "${row.domain}" und ihre Postfächer löschen?`, 'Bestätigen', { type: 'warning' })
  await http.delete(`/mail/domains/${row.id}`)
  ElMessage.success('Gelöscht')
  await load()
}

async function createAccount() {
  await http.post('/mail/accounts', {
    address: accountDialog.address,
    password: accountDialog.password,
    quota: accountDialog.quota,
  })
  ElMessage.success('Postfach erstellt')
  accountDialog.visible = false
  accountDialog.address = ''
  accountDialog.password = ''
  await load()
}

async function removeAccount(row: any) {
  await ElMessageBox.confirm(`Postfach "${row.address}" löschen?`, 'Bestätigen', { type: 'warning' })
  await http.delete(`/mail/accounts/${row.id}`)
  ElMessage.success('Gelöscht')
  await load()
}

onMounted(load)
</script>
