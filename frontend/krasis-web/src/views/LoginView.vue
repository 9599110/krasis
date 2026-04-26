<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const isRegister = ref(false)
const email = ref('')
const password = ref('')
const name = ref('')
const error = ref('')
const submitting = computed(() => authStore.loading)

async function handleSubmit() {
  error.value = ''
  try {
    if (isRegister.value) {
      if (!name.value.trim()) {
        error.value = 'Name is required'
        return
      }
      await authStore.register({
        email: email.value.trim(),
        password: password.value,
        name: name.value.trim(),
      })
    } else {
      await authStore.login({
        email: email.value.trim(),
        password: password.value,
      })
    }
    const redirect = (route.query.redirect as string) || '/app/notes'
    router.replace(redirect)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { message?: string } } }
    error.value = err.response?.data?.message || 'Authentication failed. Please try again.'
  }
}

function getOAuthUrl(provider: string) {
  const base = import.meta.env.VITE_API_BASE_URL || ''
  return `${base}/auth/oauth?provider=${provider}`
}
</script>

<template>
  <div class="login-page">
    <div class="login-container">
      <div class="login-header">
        <div class="logo-icon">
          <svg viewBox="0 0 64 64" width="40" height="40">
            <circle cx="32" cy="32" r="28" fill="var(--primary)" opacity="0.15" />
            <path d="M20 32 L28 22 L36 42 L44 32" stroke="var(--primary)" stroke-width="3" fill="none" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </div>
        <h2>{{ isRegister ? 'Create Account' : 'Welcome Back' }}</h2>
        <p class="subtitle">{{ isRegister ? 'Sign up to get started with Krasis' : 'Sign in to your account' }}</p>
      </div>

      <div class="oauth-buttons">
        <a :href="getOAuthUrl('github')" class="oauth-btn oauth-github">
          <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor"><path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/></svg>
          Continue with GitHub
        </a>
        <a :href="getOAuthUrl('google')" class="oauth-btn oauth-google">
          <svg viewBox="0 0 24 24" width="20" height="20"><path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z"/><path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/><path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/><path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/></svg>
          Continue with Google
        </a>
      </div>

      <div class="divider">
        <span>or</span>
      </div>

      <form @submit.prevent="handleSubmit" class="login-form">
        <div v-if="error" class="error-msg">{{ error }}</div>

        <div v-if="isRegister" class="form-group">
          <label for="name">Name</label>
          <input id="name" v-model="name" type="text" placeholder="Your name" required />
        </div>

        <div class="form-group">
          <label for="email">Email</label>
          <input id="email" v-model="email" type="email" placeholder="you@example.com" required />
        </div>

        <div class="form-group">
          <label for="password">Password</label>
          <input id="password" v-model="password" type="password" placeholder="Password" required minlength="6" />
        </div>

        <button type="submit" class="btn-primary" :disabled="submitting">
          {{ submitting ? 'Please wait...' : (isRegister ? 'Create Account' : 'Sign In') }}
        </button>
      </form>

      <p class="toggle-text">
        {{ isRegister ? 'Already have an account?' : "Don't have an account?" }}
        <a href="#" @click.prevent="isRegister = !isRegister">
          {{ isRegister ? 'Sign in' : 'Sign up' }}
        </a>
      </p>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: var(--bg);
  padding: 20px;
}

.login-container {
  width: 100%;
  max-width: 420px;
  background: var(--card-bg, var(--bg));
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 40px 32px;
  box-shadow: var(--shadow, 0 4px 12px rgba(0,0,0,0.08));
}

.login-header {
  text-align: center;
  margin-bottom: 28px;
}

.logo-icon {
  margin-bottom: 12px;
}

.login-header h2 {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-h);
  margin: 0 0 8px;
}

.subtitle {
  color: var(--text-muted);
  margin: 0;
  font-size: 14px;
}

.oauth-buttons {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 20px;
}

.oauth-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 10px 16px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--social-bg, rgba(108,99,255,0.05));
  color: var(--text-h);
  text-decoration: none;
  font-size: 14px;
  font-weight: 500;
  transition: background 0.2s, border-color 0.2s;
  cursor: pointer;
}

.oauth-btn:hover {
  background: var(--hover-bg, rgba(108,99,255,0.08));
  border-color: var(--primary);
}

.divider {
  display: flex;
  align-items: center;
  gap: 16px;
  margin: 20px 0;
  color: var(--text-muted);
  font-size: 13px;
}

.divider::before,
.divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--border);
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.error-msg {
  padding: 10px 14px;
  background: rgba(239,68,68,0.1);
  border: 1px solid rgba(239,68,68,0.3);
  border-radius: 6px;
  color: #ef4444;
  font-size: 13px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-group label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-h);
}

.form-group input {
  padding: 10px 14px;
  border: 1px solid var(--border);
  border-radius: 8px;
  font-size: 14px;
  background: var(--bg);
  color: var(--text-h);
  transition: border-color 0.2s;
}

.form-group input:focus {
  outline: none;
  border-color: var(--primary);
  box-shadow: 0 0 0 3px rgba(108,99,255,0.15);
}

.btn-primary {
  padding: 12px;
  border: none;
  border-radius: 8px;
  background: var(--primary);
  color: #fff;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s, opacity 0.2s;
  margin-top: 4px;
}

.btn-primary:hover:not(:disabled) {
  background: var(--primary-dark, #5a52e0);
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.toggle-text {
  text-align: center;
  margin: 24px 0 0;
  font-size: 14px;
  color: var(--text-muted);
}

.toggle-text a {
  color: var(--primary);
  text-decoration: none;
  font-weight: 500;
}

.toggle-text a:hover {
  text-decoration: underline;
}
</style>
