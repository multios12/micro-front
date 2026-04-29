import { writable } from 'svelte/store'

import { fetchBlogList } from './admin-api'
import { isProfileBlog } from '../pages/blogs/logic'

export const blogCount = writable(0)

export async function refreshBlogCount(): Promise<void> {
  try {
    const response = await fetchBlogList({ perPage: 1 })
    blogCount.set(response.items.filter((item) => !isProfileBlog(item.title)).length)
  } catch {
    blogCount.set(0)
  }
}
