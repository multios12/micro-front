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

export const dashboardBlogRows: readonly DashboardBlogRow[] = [
  {
    id: 42,
    title: '公開サイトの導線を見直す',
    summary: '公開サイトの Latest から記事一覧への導線を整理した変更です。',
    category: 'カテゴリ1',
    status: 'public',
    publishedAt: '2026-04-13',
    updatedAt: '2026-04-13 09:12',
    href: '#/blog-edit/42',
  },
  {
    id: 40,
    title: 'カテゴリ別一覧のページネーション確認',
    summary: '公開前の下書きです。',
    category: 'カテゴリ2',
    status: 'private',
    publishedAt: '2026-04-10',
    updatedAt: '2026-04-12 14:10',
    href: '#/blog-edit/40',
  },
] as const

export const dashboardSettings: readonly DashboardSettingRow[] = [
  ['ADMIN_STATIC_DIR', './web/static'],
  ['STATIC_EXPORT_DIR', './data/publish'],
  ['DATA_DIR', './data'],
  ['TOP_PAGE_BLOG_LIMIT', '20'],
  ['BLOGS_PER_PAGE', '20'],
] as const
