<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { getStatsOverview } from '../../api/admin'
import { useRouter } from 'vue-router'
import { ChartIcon, UserListIcon, UserIcon, File1Icon, ServerIcon } from 'tdesign-icons-vue-next'

const router = useRouter()
const loading = ref(true)
const stats = ref<any>({})

onMounted(async () => {
  try {
    const statsRes = await getStatsOverview()
    stats.value = statsRes.data?.data || statsRes.data || {}
  } catch (e) {
    console.error('Failed to load dashboard stats:', e)
  } finally {
    loading.value = false
  }
})

const statCards = computed(() => [
  { label: '用户总数', value: stats.value.total_users || 0, icon: UserListIcon },
  { label: '活跃用户', value: stats.value.active_users || 0, icon: UserIcon },
  { label: '笔记总数', value: stats.value.total_notes || 0, icon: File1Icon },
  { label: '存储使用', value: `${(stats.value.storage_used_gb || 0).toFixed(2)} GB`, icon: ServerIcon },
])
</script>

<template>
  <div class="dashboard">
    <div class="page-header">
      <ChartIcon class="page-icon" />
      <h1>系统概览</h1>
      <span class="page-desc">实时查看系统运行状态</span>
    </div>

    <t-skeleton :loading="loading" row-cols="4" :row-height="90" :rows="1" animation="gradient">
      <div class="stat-cards">
        <div v-for="card in statCards" :key="card.label" class="stat-card">
          <div class="stat-icon">
            <component :is="card.icon" />
          </div>
          <div class="stat-info">
            <span class="stat-value">{{ card.value }}</span>
            <span class="stat-label">{{ card.label }}</span>
          </div>
        </div>
      </div>
    </t-skeleton>
  </div>
</template>

<style scoped>
.stat-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
}
</style>
