<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const authStore = useAuthStore()

onMounted(async () => {
  const token = localStorage.getItem('auth_token')
  if (token) {
    try {
      await authStore.me()
      router.replace('/app/notes')
    } catch {
      router.replace('/login')
    }
  } else {
    router.replace('/login')
  }
})
</script>

<template>
  <div class="splash">
    <div class="splash-logo">
      <div class="logo-icon">
        <svg viewBox="0 0 64 64" width="64" height="64">
          <circle cx="32" cy="32" r="28" fill="var(--primary)" opacity="0.15" />
          <path d="M20 32 L28 22 L36 42 L44 32" stroke="var(--primary)" stroke-width="3" fill="none" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </div>
      <h1>Krasis</h1>
      <p class="tagline">Intelligent Notes System</p>
    </div>
    <div class="spinner" />
  </div>
</template>

<style scoped>
.splash {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: var(--bg);
  gap: 32px;
}

.splash-logo {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.logo-icon svg {
  width: 64px;
  height: 64px;
}

.splash-logo h1 {
  font-size: 36px;
  font-weight: 700;
  color: var(--text-h);
  margin: 0;
}

.tagline {
  font-size: 16px;
  color: var(--text-muted);
  margin: 0;
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--border);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
