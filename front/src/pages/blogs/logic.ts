export type BlogsHeaderAction = {
  label: string
  href: string
  primary?: boolean
}

export type BlogsPaginationItem = {
  label: string
  active?: boolean
  onClick?: () => void
}

export type BlogsRow = {
  id: number
  title: string
  summary: string
  category: string
  status: 'public' | 'private'
  publishedAt: string
  updatedAt: string
  href: string
}

export type BlogsFilterState = {
  status: 'all' | 'public' | 'private'
  searchValue: string
  categoryValue: string
}

export const isProfileBlog = (title: string) => title.trim().toLowerCase() === 'about'

export const blogsHeaderActions: readonly BlogsHeaderAction[] = [
  { label: 'ダッシュボードに戻る', href: '#/dashboard' },
  { label: '新規記事', href: '#/blog-edit', primary: true },
] as const

export const blogsStatusOptions = [
  { label: 'すべて', value: 'all' },
  { label: '公開', value: 'public' },
  { label: '非公開', value: 'private' },
] as const

export function mapBlogsRows(items: Array<{
  id: number
  title: string
  summary: string
  category: string
  status: 'public' | 'private'
  published_at: string
  updated_at: string
}>): BlogsRow[] {
  return items.filter((item) => !isProfileBlog(item.title)).map((item) => ({
    id: item.id,
    title: item.title,
    summary: item.summary,
    category: item.category || '未分類',
    status: item.status,
    publishedAt: item.published_at,
    updatedAt: item.updated_at,
    href: `#/blog-edit/${item.id}`,
  }))
}

export function filterBlogsRows(rows: BlogsRow[], filter: BlogsFilterState): BlogsRow[] {
  const keyword = filter.searchValue.trim().toLowerCase()
  return rows.filter((row) => {
    const matchesKeyword =
      keyword.length === 0 ||
      [row.title, row.summary, row.category].some((value) => value.toLowerCase().includes(keyword))
    const matchesCategory = filter.categoryValue === 'all' || row.category === filter.categoryValue
    const matchesStatus = filter.status === 'all' || row.status === filter.status
    return matchesKeyword && matchesCategory && matchesStatus
  })
}

export function buildBlogsPaginationItems(
  currentPage: number,
  totalPages: number,
  onPageSelect: (page: number) => void,
): BlogsPaginationItem[] {
  if (totalPages <= 1) {
    return []
  }

  const items: BlogsPaginationItem[] = []
  if (currentPage > 1) {
    items.push({ label: 'Prev', onClick: () => onPageSelect(currentPage - 1) })
  }

  const start = Math.max(1, currentPage - 1)
  const end = Math.min(totalPages, currentPage + 1)
  for (let page = start; page <= end; page += 1) {
    items.push({ label: String(page), active: page === currentPage, onClick: () => onPageSelect(page) })
  }

  if (currentPage < totalPages) {
    items.push({ label: 'Next', onClick: () => onPageSelect(currentPage + 1) })
  }

  return items
}

export function buildBlogsCountLabel(visibleCount: number, totalCount: number): string {
  return `件数: ${visibleCount} / ${totalCount}`
}
