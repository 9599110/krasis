import { ref } from 'vue'
import { defineStore } from 'pinia'
import apiClient from '../api/client'
import type { ApiResponse, SearchResult, SearchResponse } from '../api/types'

export const useSearchStore = defineStore('search', () => {
  const results = ref<SearchResult[]>([])
  const query = ref('')
  const page = ref(1)
  const total = ref(0)
  const loading = ref(false)
  const size = 20

  async function search(q: string, p = 1) {
    if (!q.trim()) {
      results.value = []
      total.value = 0
      return
    }
    loading.value = true
    query.value = q
    page.value = p
    try {
      const res = await apiClient.get<ApiResponse<SearchResponse>>('/search', {
        params: { q, page: p, size },
      })
      results.value = res.data.data.results
      total.value = res.data.data.total
      page.value = res.data.data.page
    } finally {
      loading.value = false
    }
  }

  return {
    results,
    query,
    page,
    total,
    loading,
    size,
    search,
  }
})
