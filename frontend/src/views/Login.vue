<template>
  <div class="login-wrapper">
    <el-card class="login-card">
      <h1 class="login-title">DigitarloPanel</h1>
      <p class="login-sub">Server-Administration</p>
      <el-form @submit.prevent="onSubmit">
        <el-form-item>
          <el-input v-model="username" placeholder="Benutzername" size="large" :prefix-icon="User" />
        </el-form-item>
        <el-form-item>
          <el-input
            v-model="password"
            type="password"
            placeholder="Passwort"
            size="large"
            show-password
            :prefix-icon="Lock"
            @keyup.enter="onSubmit"
          />
        </el-form-item>
        <el-button type="primary" size="large" :loading="loading" style="width: 100%" @click="onSubmit">
          Anmelden
        </el-button>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock } from '@element-plus/icons-vue'
import { useAuthStore } from '../store/auth'

const username = ref('admin')
const password = ref('')
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
    await auth.login(username.value, password.value)
    router.push({ name: 'dashboard' })
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
