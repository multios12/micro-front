export type DashboardStatCard = {
  label: string
  value: string
  note?: string
}

export type DashboardAction = {
  label: string
  href: string
  primary?: boolean
}

export type DashboardPaginationItem = {
  label: string
  active?: boolean
  onClick?: () => void
}

export type DashboardBlogRow = {
  id: number
  title: string
  summary: string
  category: string
  status: 'public' | 'private'
  publishedAt: string
  updatedAt: string
  href: string
}

export type DashboardSettingRow = readonly [string, string]

export type DashboardData = {
  blogRows: DashboardBlogRow[]
  statCards: DashboardStatCard[]
  settings: DashboardSettingRow[]
  totalPages: number
}

export const dashboardTableActions: readonly DashboardAction[] = [
  { label: '一覧ページで開く', href: '#/blogs' },
  { label: '新規記事', href: '#/blog-edit', primary: true },
] as const

export const dashboardPublishAction: DashboardAction = {
  label: '全体を再生成',
  href: '#',
  primary: true,
}

export const dashboardPageSize = 5

export function mapDashboardBlogRows(items: Array<{
  id: number
  title: string
  summary: string
  category: string
  status: 'public' | 'private'
  published_at: string
  updated_at: string
}>): DashboardBlogRow[] {
  return items.map((item) => ({
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

export function buildDashboardStatCards(items: Array<{ status: 'public' | 'private'; category: string }>): DashboardStatCard[] {
  return [
    { label: '公開中の記事', value: String(items.filter((item) => item.status === 'public').length) },
    { label: '非公開の記事', value: String(items.filter((item) => item.status === 'private').length) },
    { label: '記事総数', value: String(items.length) },
    { label: 'カテゴリ数', value: String(new Set(items.map((item) => item.category).filter(Boolean)).size) },
  ]
}

export function buildDashboardSettings(site: {
  site_title: string
  site_subtitle: string
  foot_information: string
  copyright: string
  tabs: Array<unknown>
  updated_at: string
}): DashboardSettingRow[] {
  return [
    ['サイトタイトル', site.site_title],
    ['サイトサブタイトル', site.site_subtitle],
    ['フッタ情報', site.foot_information],
    ['コピーライト', site.copyright],
    ['タブ数', String(site.tabs.length)],
    ['最終更新', site.updated_at],
  ]
}

export function buildDashboardPaginationItems(
  currentPage: number,
  totalPages: number,
  onPageSelect: (page: number) => void,
): DashboardPaginationItem[] {
  if (totalPages <= 1) {
    return []
  }

  const items: DashboardPaginationItem[] = []
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

export function createDashboardData(args: {
  blogs: Array<{
    id: number
    title: string
    summary: string
    category: string
    status: 'public' | 'private'
    published_at: string
    updated_at: string
  }>
  site: {
    site_title: string
    site_subtitle: string
    foot_information: string
    copyright: string
    tabs: Array<unknown>
    updated_at: string
  }
}): DashboardData {
  const blogRows = mapDashboardBlogRows(args.blogs)
  return {
    blogRows,
    statCards: buildDashboardStatCards(args.blogs),
    settings: buildDashboardSettings(args.site),
    totalPages: Math.max(1, Math.ceil(blogRows.length / dashboardPageSize)),
  }
}
