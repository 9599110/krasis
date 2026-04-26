<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  id: string
  title: string
  content: string
  updatedAt: string
  folderName?: string
}>()

const emit = defineEmits<{
  (e: 'edit', id: string): void
  (e: 'delete', id: string): void
  (e: 'versions', id: string): void
  (e: 'share', id: string): void
}>()

const preview = computed(() => {
  const text = props.content.replace(/[#*`_\[\]()>-]/g, '').trim()
  return text.length > 120 ? text.slice(0, 120) + '...' : text
})

const dateStr = computed(() => {
  const d = new Date(props.updatedAt)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'Just now'
  if (mins < 60) return `${mins}m ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}d ago`
  return d.toLocaleDateString()
})

function onEdit() { emit('edit', props.id) }
function onDelete() { emit('delete', props.id) }
function onVersions() { emit('versions', props.id) }
function onShare() { emit('share', props.id) }
</script>

<template>
  <div class="note-card">
    <div class="note-card-content" @click="onEdit">
      <h3 class="note-title">{{ title || 'Untitled' }}</h3>
      <p class="note-preview">{{ preview || 'Empty note' }}</p>
      <div class="note-meta">
        <span class="note-date">{{ dateStr }}</span>
        <span v-if="folderName" class="note-folder">{{ folderName }}</span>
      </div>
    </div>
    <div class="note-actions">
      <button class="action-btn" title="Versions" @click.stop="onVersions">
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M12 1v4M12 19v4M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83M1 12h4M19 12h4M4.22 19.78l2.83-2.83M16.95 7.05l2.83-2.83"/></svg>
      </button>
      <button class="action-btn" title="Share" @click.stop="onShare">
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2"><circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"/><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"/></svg>
      </button>
      <button class="action-btn action-btn--danger" title="Delete" @click.stop="onDelete">
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>
      </button>
    </div>
  </div>
</template>

<style scoped>
.note-card {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--card-bg, var(--bg));
  transition: border-color 0.2s, box-shadow 0.2s;
}

.note-card:hover {
  border-color: var(--primary);
  box-shadow: 0 2px 8px rgba(108,99,255,0.1);
}

.note-card-content {
  flex: 1;
  cursor: pointer;
  min-width: 0;
}

.note-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-h);
  margin: 0 0 6px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.note-preview {
  font-size: 13px;
  color: var(--text-muted);
  margin: 0 0 8px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-height: 1.5;
}

.note-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.note-date {
  color: var(--text-muted);
}

.note-folder {
  padding: 2px 8px;
  background: var(--accent-bg, rgba(108,99,255,0.08));
  color: var(--primary);
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
}

.note-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: background 0.2s, color 0.2s;
}

.action-btn:hover {
  background: var(--hover-bg, rgba(108,99,255,0.08));
  color: var(--text-h);
}

.action-btn--danger:hover {
  background: rgba(239,68,68,0.1);
  color: #ef4444;
}
</style>
