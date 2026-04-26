<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { MessagePlugin } from 'tdesign-vue-next'

const router = useRouter()
const authStore = useAuthStore()

const form = ref({
  email: '',
  username: '',
  password: '',
  confirmPassword: '',
})
const loading = ref(false)

async function handleRegister() {
  if (!form.value.email || !form.value.username || !form.value.password) {
    MessagePlugin.warning('请填写所有必填项')
    return
  }
  if (!form.value.email.includes('@')) {
    MessagePlugin.warning('邮箱格式不正确')
    return
  }
  if (form.value.password.length < 6) {
    MessagePlugin.warning('密码至少 6 位')
    return
  }
  if (form.value.password !== form.value.confirmPassword) {
    MessagePlugin.warning('两次密码不一致')
    return
  }

  loading.value = true
  try {
    await authStore.register({
      email: form.value.email.trim(),
      password: form.value.password,
      name: form.value.username.trim(),
    })
    MessagePlugin.success('注册成功')
    router.push({ name: 'notes' })
  } catch (e: any) {
    MessagePlugin.error(e.response?.data?.message || '注册失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="register-page">
    <div class="register-card">
      <div class="card-header">
        <t-icon name="layers" size="28px" style="color: #1677ff" />
        <h1>Krasis</h1>
      </div>
      <h2>注册账号</h2>

      <div class="form-group">
        <label>邮箱</label>
        <t-input
          v-model="form.email"
          type="email"
          name="email"
          autocomplete="email"
          placeholder="请输入邮箱"
          clearable
          @enter="handleRegister"
        />
      </div>

      <div class="form-group">
        <label>昵称</label>
        <t-input
          v-model="form.username"
          name="username"
          autocomplete="name"
          placeholder="请输入昵称"
          clearable
          @enter="handleRegister"
        />
      </div>

      <div class="form-group">
        <label>密码</label>
        <t-input
          v-model="form.password"
          type="password"
          name="new-password"
          autocomplete="new-password"
          placeholder="至少 6 位"
          @enter="handleRegister"
        />
      </div>

      <div class="form-group">
        <label>确认密码</label>
        <t-input
          v-model="form.confirmPassword"
          type="password"
          name="confirm-password"
          autocomplete="new-password"
          placeholder="再次输入密码"
          @enter="handleRegister"
        />
      </div>

      <t-button block @click="handleRegister" :loading="loading" style="margin-top: 8px">
        注册
      </t-button>

      <div class="footer-links">
        <router-link :to="{ name: 'login' }">已有账号？返回登录</router-link>
      </div>
    </div>
  </div>
</template>

<style scoped>
.register-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: #f7f8fa;
}

.register-card {
  width: 400px;
  background: #fff;
  border-radius: 12px;
  padding: 40px 32px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.06);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 24px;
}

.card-header h1 {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  color: #1d2129;
}

h2 {
  margin: 0 0 24px;
  font-size: 18px;
  color: #4e5969;
  font-weight: 500;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 16px;
}

.form-group label {
  font-size: 13px;
  font-weight: 500;
  color: #4e5969;
}

.footer-links {
  margin-top: 16px;
  text-align: center;
}

.footer-links a {
  font-size: 13px;
  color: #1677ff;
  text-decoration: none;
}

.footer-links a:hover {
  text-decoration: underline;
}
</style>
