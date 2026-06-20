<template>
  <div class="login-wrapper">
    <el-card class="login-card">
      <h1 class="login-title">DigitarloPanel</h1>
      <p class="login-sub">Server-Administration</p>
      <el-form @submit.prevent="onSubmit">
        <el-form-item>
          <el-input v-model="username" placeholder="Benutzername" size="large" :prefix-icon="User" :disabled="needCode" />
        </el-form-item>
        <el-form-item>
          <el-input
            v-model="password"
            type="password"
            placeholder="Passwort"
            size="large"
            show-password
            :prefix-icon="Lock"
            :disabled="needCode"
            @keyup.enter="onSubmit"
          />
        </el-form-item>
        <el-form-item v-if="needCode">
          <el-input
            v-model="code"
            placeholder="2FA-Code (6-stellig)"
            size="large"
            maxlength="6"
            :prefix-icon="Key"
            @keyup.enter="onSubmit"
          />
        </el-form-item>
        <el-button type="primary" size="large" :loading="loading" style="width: 100%" @click="onSubmit">
          {{ needCode ? 'Code bestätigen' : 'Anmelden' }}
        </el-button>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock, Key } from '@element-plus/icons-vue'
import { useAuthStore } from '../store/auth'

const username = ref('admin')
const password = ref('')
const code = ref('')
const needCode = ref(false)
const loading = ref(false)
const auth = useAuthStore()
const router = useRouter()

async function onSubmit() {
  if (!username.value || !password.value) {
    ElMessage.warning('Bitte Benutzername und Passwort eingeben')
    return
  }
  loading.value = true
  try {
    const ok = await auth.login(username.value, password.value, code.value || undefined)
    if (ok) {
      router.push({ name: 'dashboard' })
    } else {
      needCode.value = true
      ElMessage.info('Bitte 2FA-Code eingeben')
    }
  } catch {
    // error toast handled by interceptor
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-wrapper {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1f2937, #111827);
}
.login-card {
  width: 360px;
}
.login-title {
  margin: 0;
  text-align: center;
  color: #409eff;
}
.login-sub {
  margin: 4px 0 24px;
  text-align: center;
  color: #909399;
}
</style>
