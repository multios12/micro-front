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
