<template>
  <div>
    <h2 class="page-title">Einstellungen</h2>
    <el-card shadow="never" style="max-width: 520px">
      <template #header>Passwort ändern</template>
      <el-form label-width="160px">
        <el-form-item label="Aktuelles Passwort">
          <el-input v-model="form.old" type="password" show-password />
        </el-form-item>
        <el-form-item label="Neues Passwort">
          <el-input v-model="form.new" type="password" show-password />
        </el-form-item>
        <el-form-item label="Wiederholen">
          <el-input v-model="form.confirm" type="password" show-password />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="saving" @click="submit">Speichern</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api/client'

const form = reactive({ old: '', new: '', confirm: '' })
const saving = ref(false)

async function submit() {
  if (form.new.length < 8) {
    ElMessage.warning('Neues Passwort muss mindestens 8 Zeichen haben')
    return
  }
  if (form.new !== form.confirm) {
    ElMessage.warning('Passwörter stimmen nicht überein')
    return
  }
  saving.value = true
  try {
    await http.post('/change-password', { old_password: form.old, new_password: form.new })
    ElMessage.success('Passwort geändert')
    form.old = ''
    form.new = ''
    form.confirm = ''
  } finally {
    saving.value = false
  }
}
</script>
