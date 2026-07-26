import { writable } from 'svelte/store'

import { fetchBlogList } from './admin-api'

export const blogCount = writable(0)

export async function refreshBlogCount(): Promise<void> {
  try {
    const response = await fetchBlogList({ perPage: 1 })
    blogCount.set(response.total)
  } catch {
    blogCount.set(0)
  }
}
