<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { getStatsOverview, getShareStats } from '../../api/admin'
import { useRouter } from 'vue-router'
import { ChartIcon, UserListIcon, UserIcon, File1Icon, Share1Icon, ServerIcon } from 'tdesign-icons-vue-next'

const router = useRouter()
const loading = ref(true)
const stats = ref<any>({})
const shareStats = ref<any>({ pending: 0, approved: 0, rejected: 0 })

onMounted(async () => {
  try {
    const [statsRes, shareRes] = await Promise.allSettled([
      getStatsOverview(),
      getShareStats(),
    ])
    if (statsRes.status === 'fulfilled') {
      stats.value = statsRes.value.data?.data || statsRes.value.data || {}
    }
    if (shareRes.status === 'fulfilled') {
      const d = shareRes.value.data?.data || shareRes.value.data || {}
      shareStats.value = d
    }
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
  { label: '分享总数', value: stats.value.total_shares || 0, icon: Share1Icon },
  { label: '存储使用', value: `${(stats.value.storage_used_gb || 0).toFixed(2)} GB`, icon: ServerIcon },
])

function goToShares() {
  router.push({ name: 'admin-shares' })
}
</script>

<template>
  <div class="dashboard">
    <div class="page-header">
      <ChartIcon class="page-icon" />
      <h1>系统概览</h1>
      <span class="page-desc">实时查看系统运行状态</span>
    </div>

    <t-skeleton :loading="loading" row-cols="5" :row-height="90" :rows="1" animation="gradient">
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

    <div class="section" style="margin-top: 24px;">
      <t-card title="待审核分享" :bordered="true" hover>
        <template #actions>
          <t-button size="small" @click="goToShares">查看全部</t-button>
        </template>
        <div class="share-stats">
          <div class="stat-mini">
            <span class="stat-mini-count pending">{{ shareStats.pending || 0 }}</span>
            <span class="stat-mini-label">待审核</span>
          </div>
          <div class="stat-mini">
            <span class="stat-mini-count approved">{{ shareStats.approved || 0 }}</span>
            <span class="stat-mini-label">已通过</span>
          </div>
          <div class="stat-mini">
            <span class="stat-mini-count rejected">{{ shareStats.rejected || 0 }}</span>
            <span class="stat-mini-label">已拒绝</span>
          </div>
          <div class="stat-mini">
            <span class="stat-mini-count revoked">{{ shareStats.revoked || 0 }}</span>
            <span class="stat-mini-label">已撤回</span>
          </div>
        </div>
      </t-card>
    </div>
  </div>
</template>

<style scoped>
.stat-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
}

.share-stats {
  display: flex;
  gap: 16px;
  padding: 12px 0;
  flex-wrap: wrap;
}
</style>
